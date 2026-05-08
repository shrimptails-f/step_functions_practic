package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type WorkerInput struct {
	WorkerID   string          `json:"workerId"`
	Payload    json.RawMessage `json:"payload"`
	RetryCount int             `json:"retryCount"`
	RetryLimit int             `json:"retryLimit"`
	JobID      string          `json:"jobId"`
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

	if input.WorkerID == "w2" && input.RetryCount == 0 {
		fmt.Fprintln(os.Stderr, "forced retryable failure for worker w2 first attempt")
		os.Exit(1)
	}

	if input.WorkerID == "w3" {
		fmt.Fprintln(os.Stderr, "forced failure for worker w3")
		os.Exit(1)
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

	printResultAndExit(Result{
		Status:           "SUCCEEDED",
		ErrorType:        "",
		Message:          "processed products and queued messages",
		ProductTotal:     2,
		ProductSucceeded: productSucceeded,
		QueuedCount:      queuedCount,
		WorkerID:         input.WorkerID,
		Payload:          input.Payload,
	})
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

func printResultAndExit(result Result) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(result)
	os.Exit(0)
}
