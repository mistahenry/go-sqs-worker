// cmd/worker/main.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"go-sqs-worker/internal/config"
	"go-sqs-worker/internal/worker"
)

func main() {
	cfg, err := config.Load(config.OSEnv{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("config ok: region=%s endpoint=%s queue=%s concurrency=%d\n",
		cfg.AWSRegion, cfg.SQSEndpoint, cfg.QueueURL, cfg.Concurrency)

	ctx := context.Background()
	client := newSQSClient(ctx, cfg)
	poller := worker.NewPoller(client, cfg.QueueURL)

	handler := func(ctx context.Context, msg *worker.Message) error {
		fmt.Printf("processing: id=%s body=%s\n", msg.ID, msg.Body)
		return nil
	}

	runner := worker.NewRunner(poller, handler, cfg.MaxInFlight, cfg.Concurrency)
	if err := runner.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newSQSClient(ctx context.Context, cfg config.Config) worker.SQSClient {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}

	if cfg.SQSEndpoint != "" {
		opts = append(opts,
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.AWSAccessKey, cfg.AWSSecretKey, ""),
			),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		panic(err)
	}

	clientOpts := []func(*sqs.Options){}
	if cfg.SQSEndpoint != "" {
		clientOpts = append(clientOpts, func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(cfg.SQSEndpoint)
		})
	}

	return sqs.NewFromConfig(awsCfg, clientOpts...)
}
