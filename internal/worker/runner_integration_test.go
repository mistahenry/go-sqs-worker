//go:build integration
// +build integration

package worker_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"go-sqs-worker/internal/worker"
)

func TestRunner_Integration_ProcessesAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Hard timeout so the test can’t hang forever.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("dummy", "dummy", ""),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9324")
	})

	_, err = client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		t.Fatalf("elasticmq unreachable: %v", err)
	}

	qName := fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
	createOut, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(qName),
	})
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	queueURL := aws.ToString(createOut.QueueUrl)

	defer func() {
		_, _ = client.DeleteQueue(context.Background(), &sqs.DeleteQueueInput{
			QueueUrl: aws.String(queueURL),
		})
	}()

	// Seed messages with a unique run MessageID so we don’t need purge.
	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	const n = 5

	for i := 1; i <= n; i++ {
		body := fmt.Sprintf(`{"run":"%s","id":"%d"}`, runID, i)
		_, err := client.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    &queueURL,
			MessageBody: &body,
		})
		if err != nil {
			t.Fatalf("send message %d: %v", i, err)
		}
	}

	poller := worker.NewPoller(client, queueURL).WithWaitTimeSeconds(5)

	var processed atomic.Int32
	var wg sync.WaitGroup
	wg.Add(n)

	handler := func(ctx context.Context, msg *worker.Message) error {
		// Only count messages from this test run.
		t.Log("Handler invoked")
		if !contains(msg.Body, runID) {
			return nil
		}
		processed.Add(1)
		wg.Done()
		return nil
	}

	runner := worker.NewRunner(poller, handler, 3, 2)

	runCtx, stop := context.WithCancel(ctx)
	defer stop()

	// Stop runner as soon as we’ve seen all expected messages.
	go func() {
		wg.Wait()
		stop()
	}()

	err = runner.Run(runCtx)
	// Runner should stop due to cancel; treat that as success.
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("runner error: %v", err)
	}

	if got := processed.Load(); got != n {
		t.Fatalf("expected %d processed, got %d", n, got)
	}
}

// keep this dumb-simple so we don’t pull in json parsing yet
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && index(s, sub) >= 0)
}

// minimal string search to avoid importing strings if you care; or just use strings.Contains
func index(s, sub string) int {
	// simplest: just use strings.Contains in real code
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
