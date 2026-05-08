package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type payload struct {
	ForceFail bool `json:"forceFail"`
}

func handler(_ context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	failures := make([]events.SQSBatchItemFailure, 0)

	for _, record := range event.Records {
		log.Printf("messageId=%s body=%s", record.MessageId, record.Body)

		var p payload
		if err := json.Unmarshal([]byte(record.Body), &p); err != nil {
			continue
		}

		if p.ForceFail {
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}

	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func main() {
	lambda.Start(handler)
}
