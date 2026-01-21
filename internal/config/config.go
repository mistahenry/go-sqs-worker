package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type EnvReader interface {
	Getenv(key string) string
}

// OSEnv reads from the actual operating system environment
type OSEnv struct{}

func (OSEnv) Getenv(key string) string {
	return os.Getenv(key)
}

type Config struct {
	AWSRegion    string
	SQSEndpoint  string
	QueueURL     string
	AWSAccessKey string
	AWSSecretKey string
	Concurrency  int
}

func Load(env EnvReader) (Config, error) {
	queueURL := env.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		return Config{}, errors.New("SQS_QUEUE_URL is required")
	}

	concurrency, err := getenvInt(env, "WORKER_CONCURRENCY", 4)
	if err != nil {
		return Config{}, err
	}
	if concurrency <= 0 {
		return Config{}, errors.New("WORKER_CONCURRENCY must be > 0")
	}

	region := getenv(env, "AWS_REGION", "us-east-1")
	endpoint := env.Getenv("SQS_ENDPOINT")
	accessKey := getenv(env, "AWS_ACCESS_KEY_ID", "dummy")
	secretKey := getenv(env, "AWS_SECRET_ACCESS_KEY", "dummy")

	return Config{
		AWSRegion:    region,
		SQSEndpoint:  endpoint,
		QueueURL:     queueURL,
		Concurrency:  concurrency,
		AWSAccessKey: accessKey,
		AWSSecretKey: secretKey,
	}, nil
}

func getenv(env EnvReader, key, def string) string {
	v := env.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getenvInt(env EnvReader, key string, def int) (int, error) {
	v := getenv(env, key, "")
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", key, v)
	}
	return n, nil
}
