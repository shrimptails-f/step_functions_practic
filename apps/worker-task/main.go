package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type WorkerInput struct {
	WorkerID    string          `json:"workerId"`
	Payload     json.RawMessage `json:"payload"`
	RetryCount  int             `json:"retryCount"`
	RetryLimit  int             `json:"retryLimit"`
	JobID       string          `json:"jobId"`
	ResultS3Key string          `json:"resultS3Key"`
}

type Result struct {
	Status           string          `json:"status"`
	ErrorType        string          `json:"errorType"`
	Message          string          `json:"message"`
	ProductTotal     int             `json:"productTotal"`
	ProductSucceeded int             `json:"productSucceeded"`
	QueuedCount      int             `json:"queuedCount"`
	WorkerID         string          `json:"workerId"`
	Payload          json.RawMessage `json:"payload"`
}

func main() {
	input, err := readInputFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("invalid WORKER_INPUT: %v", err))
		os.Exit(1)
	}
	if input.ResultS3Key == "" {
		fmt.Fprintln(os.Stderr, "WORKER_INPUT.resultS3Key is required")
		os.Exit(1)
	}
	resultsBucket := os.Getenv("RESULTS_S3_BUCKET")
	if resultsBucket == "" {
		fmt.Fprintln(os.Stderr, "RESULTS_S3_BUCKET is required")
		os.Exit(1)
	}

	if result, failed := buildFailureResultByPoCRule(input); failed {
		if err := putResultToS3(context.Background(), resultsBucket, input.ResultS3Key, result); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("failed to save worker result to s3: %v", err))
			os.Exit(1)
		}
		printResultAndExit(result)
		return
	}

	productSucceeded := processProductsConcurrently()
	if productSucceeded != 2 {
		fmt.Fprintln(os.Stderr, "product processing failed")
		os.Exit(1)
	}

	queuedCount, queueErr := sendMessagesToSQS(input)
	if queueErr != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("failed to queue messages: %v", queueErr))
		os.Exit(1)
	}

	successResult := Result{
		Status:           "SUCCEEDED",
		ErrorType:        "",
		Message:          "processed products and queued messages",
		ProductTotal:     2,
		ProductSucceeded: productSucceeded,
		QueuedCount:      queuedCount,
		WorkerID:         input.WorkerID,
		Payload:          input.Payload,
	}
	if err := putResultToS3(context.Background(), resultsBucket, input.ResultS3Key, successResult); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("failed to save worker result to s3: %v", err))
		os.Exit(1)
	}
	printResultAndExit(successResult)
}

func buildFailureResultByPoCRule(input WorkerInput) (Result, bool) {
	if input.WorkerID == "w2" && input.RetryCount == 0 {
		return Result{
			Status:           "FAILED",
			ErrorType:        "RETRYABLE",
			Message:          "forced retryable failure for worker w2 first attempt",
			ProductTotal:     2,
			ProductSucceeded: 0,
			QueuedCount:      0,
			WorkerID:         input.WorkerID,
			Payload:          input.Payload,
		}, true
	}
	if input.WorkerID == "w3" {
		return Result{
			Status:           "FAILED",
			ErrorType:        "NON_RETRYABLE",
			Message:          "forced non-retryable failure for worker w3",
			ProductTotal:     2,
			ProductSucceeded: 0,
			QueuedCount:      0,
			WorkerID:         input.WorkerID,
			Payload:          input.Payload,
		}, true
	}
	return Result{}, false
}

func readInputFromEnv() (WorkerInput, error) {
	raw := os.Getenv("WORKER_INPUT")
	if raw == "" {
		return WorkerInput{}, fmt.Errorf("WORKER_INPUT is required")
	}

	var in WorkerInput
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return WorkerInput{}, err
	}

	return in, nil
}

func processProductsConcurrently() int {
	products := []string{"product-1", "product-2"}
	var wg sync.WaitGroup
	results := make(chan bool, len(products))

	for _, product := range products {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			_ = p
			time.Sleep(10 * time.Second)
			results <- true
		}(product)
	}

	wg.Wait()
	close(results)

	succeeded := 0
	for ok := range results {
		if ok {
			succeeded++
		}
	}

	return succeeded
}

func sendMessagesToSQS(input WorkerInput) (int, error) {
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		return 0, fmt.Errorf("SQS_QUEUE_URL is required")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return 0, fmt.Errorf("load aws config: %w", err)
	}

	client := sqs.NewFromConfig(cfg)

	products := []string{"product-1", "product-2"}
	queued := 0
	for _, product := range products {
		body, err := json.Marshal(map[string]any{
			"jobId":    input.JobID,
			"workerId": input.WorkerID,
			"product":  product,
			"payload":  input.Payload,
		})
		if err != nil {
			return queued, fmt.Errorf("marshal message body: %w", err)
		}

		_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    aws.String(queueURL),
			MessageBody: aws.String(string(body)),
		})
		if err != nil {
			return queued, fmt.Errorf("send message for %s: %w", product, err)
		}

		queued++
	}

	return queued, nil
}

func putResultToS3(ctx context.Context, bucket, key string, result Result) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg)

	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func printResultAndExit(result Result) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(result)
	os.Exit(0)
}
