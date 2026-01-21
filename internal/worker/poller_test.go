package worker

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type fakeSQS struct {
	messages []types.Message
	err      error
}

func (f *fakeSQS) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &sqs.ReceiveMessageOutput{Messages: f.messages}, nil
}

func (f *fakeSQS) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

func TestPoller_ReceiveOne_ReturnsMessage(t *testing.T) {
	t.Parallel()

	msgID := "test-123"
	msgBody := "hello world"

	client := &fakeSQS{
		messages: []types.Message{
			{MessageId: &msgID, Body: &msgBody},
		},
	}

	p := NewPoller(client, "http://example.com/queue")
	msg, err := p.ReceiveOne(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.ID != msgID {
		t.Errorf("expected ID %q, got %q", msgID, msg.ID)
	}
	if msg.Body != msgBody {
		t.Errorf("expected Body %q, got %q", msgBody, msg.Body)
	}
}

func TestPoller_ReceiveOne_NoMessages(t *testing.T) {
	t.Parallel()

	client := &fakeSQS{messages: []types.Message{}}

	p := NewPoller(client, "http://example.com/queue")
	msg, err := p.ReceiveOne(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != nil {
		t.Errorf("expected nil, got %+v", msg)
	}
}
