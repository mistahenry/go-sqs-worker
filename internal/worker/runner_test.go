// internal/worker/runner_test.go
package worker

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type fakeSQS struct {
	mu             sync.Mutex
	messages       []types.Message
	nextIndex      int
	deletedHandles map[string]bool
	inFlight       int32
	maxInFlight    int32
	err            error

	// Hooks - set by individual tests
	OnReceive func(msg types.Message)
	OnDelete  func(handle string)
}

func (f *fakeSQS) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.mu.Lock()
	if f.nextIndex >= len(f.messages) {
		f.mu.Unlock()
		return &sqs.ReceiveMessageOutput{Messages: nil}, nil
	}

	msg := f.messages[f.nextIndex]
	f.nextIndex++
	f.mu.Unlock()

	current := atomic.AddInt32(&f.inFlight, 1)
	for {
		maxSeenInFlight := atomic.LoadInt32(&f.maxInFlight)
		if current <= maxSeenInFlight || atomic.CompareAndSwapInt32(&f.maxInFlight, maxSeenInFlight, current) {
			break
		}
	}

	if f.OnReceive != nil {
		f.OnReceive(msg)
	}

	return &sqs.ReceiveMessageOutput{Messages: []types.Message{msg}}, nil
}

func (f *fakeSQS) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	f.mu.Lock()
	if f.deletedHandles == nil {
		f.deletedHandles = make(map[string]bool)
	}
	deleteHandle := ""
	if params.ReceiptHandle != nil {
		deleteHandle = *params.ReceiptHandle
		f.deletedHandles[deleteHandle] = true
	}
	f.mu.Unlock()

	atomic.AddInt32(&f.inFlight, -1)

	if f.OnDelete != nil {
		f.OnDelete(deleteHandle)
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (f *fakeSQS) GetMaxInFlight() int32 {
	return atomic.LoadInt32(&f.maxInFlight)
}

func (f *fakeSQS) GetDeletedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.deletedHandles)
}

func makeMessages(n int) []types.Message {
	var messages []types.Message
	for i := 1; i <= n; i++ {
		id := strconv.Itoa(i)
		messages = append(messages, types.Message{
			MessageId:     &id,
			Body:          &id,
			ReceiptHandle: &id,
		})
	}
	return messages
}

func TestRunner_WorkerPool_ProcessesConcurrently(t *testing.T) {
	const numWorkers = 3
	const numMessages = 10

	var (
		currentConcurrent atomic.Int32
		maxConcurrent     atomic.Int32
		atBarrier         atomic.Int32
		allAtBarrier      = make(chan struct{})
		release           = make(chan struct{})
		allDeleted        = make(chan struct{})
		deleted           atomic.Int32
	)

	client := &fakeSQS{
		messages: makeMessages(numMessages),
		OnDelete: func(handle string) {
			if deleted.Add(1) == int32(numMessages) {
				close(allDeleted)
			}
		},
	}
	poller := NewPoller(client, "http://example.com/queue")

	handler := func(ctx context.Context, msg *Message) error {
		current := currentConcurrent.Add(1)
		defer currentConcurrent.Add(-1)

		for {
			maxConcurrentVal := maxConcurrent.Load()
			if current <= maxConcurrentVal || maxConcurrent.CompareAndSwap(maxConcurrentVal, current) {
				break
			}
		}

		n := atBarrier.Add(1)
		if n == int32(numWorkers) {
			close(allAtBarrier)
		}
		if n <= int32(numWorkers) {
			<-release
		}

		return nil
	}

	runner := NewRunner(poller, handler, 10, numWorkers)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	// Wait for all workers to be blocked in handlers
	select {
	case <-allAtBarrier:
	case <-ctx.Done():
		t.Fatal("timeout waiting for workers to reach barrier")
	}

	// At this point, exactly numWorkers handlers are running
	if maxConcurrentSeen := maxConcurrent.Load(); maxConcurrentSeen != int32(numWorkers) {
		t.Errorf("max concurrent = %d, want exactly %d", maxConcurrentSeen, numWorkers)
	}

	// Release the barrier, let remaining messages process
	close(release)

	select {
	case <-allDeleted:
	case <-ctx.Done():
		t.Fatal("timeout waiting for messages to process")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for runner to exit")
	}
}

func TestRunner_RespectsMaxInFlight(t *testing.T) {
	const maxInFlight = 5
	const numWorkers = 3
	const numMessages = 20

	exceeded := make(chan struct{})
	allDeleted := make(chan struct{})

	var once sync.Once
	var deleted atomic.Int32

	var client *fakeSQS
	client = &fakeSQS{
		messages: makeMessages(numMessages),
		OnReceive: func(msg types.Message) {
			if client.GetMaxInFlight() > int32(maxInFlight) {
				once.Do(func() { close(exceeded) })
			}
		},
		OnDelete: func(handle string) {
			if deleted.Add(1) == int32(numMessages) {
				close(allDeleted)
			}
		},
	}
	poller := NewPoller(client, "http://example.com/queue")

	handler := func(ctx context.Context, msg *Message) error {
		return nil
	}

	runner := NewRunner(poller, handler, maxInFlight, numWorkers)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	select {
	case <-exceeded:
		t.Fatalf("exceeded max in-flight: got %d, want <= %d", client.GetMaxInFlight(), maxInFlight)
	case <-allDeleted:
		// Good - processed all messages without exceeding
	case <-ctx.Done():
		t.Fatal("timeout waiting for messages to process")
	}

	// clean up runner to avoid leakage
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for runner to exit")
	}
}

func TestRunner_ProcessesAllMessages(t *testing.T) {
	const numMessages = 10

	var processed atomic.Int32
	allDeleted := make(chan struct{})
	var deleted atomic.Int32

	client := &fakeSQS{
		messages: makeMessages(numMessages),
		OnDelete: func(handle string) {
			if deleted.Add(1) == int32(numMessages) {
				close(allDeleted)
			}
		},
	}
	poller := NewPoller(client, "http://example.com/queue")

	handler := func(ctx context.Context, msg *Message) error {
		processed.Add(1)
		return nil
	}

	runner := NewRunner(poller, handler, 5, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	select {
	case <-allDeleted:
		// Good
	case <-ctx.Done():
		t.Fatal("timeout waiting for all messages to process")
	}

	if got := processed.Load(); got != numMessages {
		t.Errorf("processed %d messages, want %d", got, numMessages)
	}

	if got := client.GetDeletedCount(); got != numMessages {
		t.Errorf("deleted %d messages, want %d", got, numMessages)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for runner to exit")
	}
}
