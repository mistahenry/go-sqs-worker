package main

import (
	"fmt"
	"os"

	"go-sqs-worker/internal/config"
)

func main() {
	cfg, err := config.Load(config.OSEnv{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("config ok: region=%s endpoint=%s queue=%s concurrency=%d\n",
		cfg.AWSRegion, cfg.SQSEndpoint, cfg.QueueURL, cfg.Concurrency)
}
