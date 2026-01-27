package worker

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Poller struct {
	client          SQSClient
	queueURL        string
	waitTimeSeconds int32
}

type Message struct {
	MessageID     string
	Body          string
	ReceiptHandle *string
}

type Handler func(ctx context.Context, msg *Message) error

func NewPoller(client SQSClient, queueURL string) *Poller {
	return &Poller{
		client:          client,
		queueURL:        queueURL,
		waitTimeSeconds: 5,
	}
}

func (p *Poller) ProcessOne(ctx context.Context, handler Handler) error {
	msg, err := p.ReceiveOne(ctx)
	if err != nil {
		return err
	}

	if msg == nil {
		return nil // no messages
	}

	// Call handler - if it succeeds, delete
	if err := handler(ctx, msg); err != nil {
		return fmt.Errorf("handler: %w", err)
	}

	// Delete only on success
	if err := p.Delete(ctx, msg); err != nil {
		return err
	}

	return nil
}

func (p *Poller) Delete(ctx context.Context, msg *Message) error {
	_, err := p.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &p.queueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
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
		MessageID:     *msg.MessageId,
		Body:          *msg.Body,
		ReceiptHandle: msg.ReceiptHandle,
	}, nil
}

func (p *Poller) WithWaitTimeSeconds(seconds int32) *Poller {
	p.waitTimeSeconds = seconds
	return p
}
