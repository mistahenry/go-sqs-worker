package worker

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Poller struct {
	client   SQSClient
	queueURL string
}

type Message struct {
	ID   string
	Body string
}

func NewPoller(client SQSClient, queueURL string) *Poller {
	return &Poller{
		client:   client,
		queueURL: queueURL,
	}
}

func (p *Poller) ReceiveOne(ctx context.Context) (*Message, error) {
	out, err := p.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &p.queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     5,
	})
	if err != nil {
		return nil, fmt.Errorf("receive: %w", err)
	}

	if len(out.Messages) == 0 {
		return nil, nil
	}

	msg := out.Messages[0]
	return &Message{
		ID:   *msg.MessageId,
		Body: *msg.Body,
	}, nil
}
