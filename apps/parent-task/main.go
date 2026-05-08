package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type stepFunctionsInput struct {
	BatchID string `json:"batchId"`
	Text    string `json:"text"`
}

type workerPayload struct {
	Value int `json:"value"`
}

type worker struct {
	WorkerID string        `json:"workerId"`
	Payload  workerPayload `json:"payload"`
}

func parseInput() (stepFunctionsInput, error) {
	raw := os.Getenv("STEP_FUNCTIONS_INPUT")
	if raw == "" {
		return stepFunctionsInput{}, errors.New("STEP_FUNCTIONS_INPUT is required")
	}

	var in stepFunctionsInput
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return stepFunctionsInput{}, fmt.Errorf("failed to parse STEP_FUNCTIONS_INPUT: %w", err)
	}

	if in.BatchID == "" {
		return stepFunctionsInput{}, errors.New("STEP_FUNCTIONS_INPUT.batchId is required")
	}
	if in.Text == "" {
		return stepFunctionsInput{}, errors.New("STEP_FUNCTIONS_INPUT.text is required")
	}

	return in, nil
}

func buildWorkers() []worker {
	return []worker{
		{WorkerID: "w1", Payload: workerPayload{Value: 1}},
		{WorkerID: "w2", Payload: workerPayload{Value: 2}},
		{WorkerID: "w3", Payload: workerPayload{Value: 3}},
	}
}

func writeWorkersToS3(ctx context.Context, bucket, key string, workers []worker) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &bucket, Key: &key})
	if err == nil {
		return fmt.Errorf("workers object already exists for this batch: s3://%s/%s", bucket, key)
	}
	var apiErr smithy.APIError
	if !(errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "404")) {
		return fmt.Errorf("head object failed: %w", err)
	}

	body, err := json.Marshal(workers)
	if err != nil {
		return fmt.Errorf("marshal workers json: %w", err)
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(body),
		ContentType: awsString("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put object failed: %w", err)
	}

	return nil
}
func awsString(s string) *string { return &s }

func main() {
	_, err := parseInput()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	bucket := os.Getenv("WORKERS_S3_BUCKET")
	key := os.Getenv("WORKERS_S3_KEY")
	if bucket == "" || key == "" {
		fmt.Fprintln(os.Stderr, "WORKERS_S3_BUCKET and WORKERS_S3_KEY are required")
		os.Exit(1)
	}

	time.Sleep(10 * time.Second)

	if err := writeWorkersToS3(context.Background(), bucket, key, buildWorkers()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
