package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type fakeSQS struct {
	messages       []types.Message
	deletedHandles map[string]bool
	err            error
	mu             sync.Mutex
}

func (f *fakeSQS) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &sqs.ReceiveMessageOutput{Messages: f.messages}, nil
}

func (f *fakeSQS) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deletedHandles == nil {
		f.deletedHandles = make(map[string]bool)
	}
	if params.ReceiptHandle != nil {
		f.deletedHandles[*params.ReceiptHandle] = true
	}
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

func TestPoller_ProcessOne_DeletesOnSuccess(t *testing.T) {
	t.Parallel()

	msgID := "test-123"
	msgBody := "hello"
	receiptHandle := "receipt-abc"

	client := &fakeSQS{
		messages: []types.Message{
			{
				MessageId:     &msgID,
				Body:          &msgBody,
				ReceiptHandle: &receiptHandle,
			},
		},
	}

	var processedMsg *Message
	handler := func(ctx context.Context, msg *Message) error {
		processedMsg = msg
		return nil // success
	}

	p := NewPoller(client, "http://example.com/queue")
	err := p.ProcessOne(context.Background(), handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processedMsg == nil {
		t.Fatal("handler was not called")
	}
	if processedMsg.ID != msgID {
		t.Errorf("expected handler to receive ID %q, got %q", msgID, processedMsg.ID)
	}
	if !client.deletedHandles[receiptHandle] {
		t.Errorf("expected receipt %q to be deleted", receiptHandle)
	}
}

func TestPoller_ProcessOne_NoDeleteOnHandlerError(t *testing.T) {
	t.Parallel()

	msgID := "test-123"
	msgBody := "hello"
	receiptHandle := "receipt-abc"

	client := &fakeSQS{
		messages: []types.Message{
			{
				MessageId:     &msgID,
				Body:          &msgBody,
				ReceiptHandle: &receiptHandle,
			},
		},
	}

	handler := func(ctx context.Context, msg *Message) error {
		return fmt.Errorf("processing failed")
	}

	p := NewPoller(client, "http://example.com/queue")
	err := p.ProcessOne(context.Background(), handler)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client.deletedHandles[receiptHandle] {
		t.Error("message should not be deleted on handler error")
	}
}
