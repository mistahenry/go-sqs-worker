package config

import (
	"testing"
)

type fakeEnv map[string]string

func (e fakeEnv) Getenv(key string) string {
	return e[key]
}

func TestLoad_MissingQueueURLFails(t *testing.T) {
	env := fakeEnv{}
	_, err := Load(env)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_DefaultsApply(t *testing.T) {
	env := fakeEnv{
		"SQS_QUEUE_URL": "http://localhost:9324/000000000000/local-sqs-worker",
	}
	cfg, err := Load(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AWSRegion != "us-east-1" {
		t.Fatalf("expected default AWSRegion us-east-1, got %q", cfg.AWSRegion)
	}
	if cfg.Concurrency != 4 {
		t.Fatalf("expected default Concurrency 4, got %d", cfg.Concurrency)
	}
	if cfg.AWSSecretKey != "dummy" {
		t.Fatalf("expected AWSSecretKey dummy, got %q", cfg.AWSSecretKey)
	}
	if cfg.AWSAccessKey != "dummy" {
		t.Fatalf("expected AWSAccessKey dummy, got %q", cfg.AWSAccessKey)
	}
}

func TestLoad_ExplicitValuesOverrideDefaults(t *testing.T) {
	env := fakeEnv{
		"SQS_QUEUE_URL":         "http://example.com/queue",
		"AWS_REGION":            "eu-west-1",
		"WORKER_CONCURRENCY":    "8",
		"SQS_ENDPOINT":          "http://localhost:9324",
		"AWS_ACCESS_KEY_ID":     "someAccessKey",
		"AWS_SECRET_ACCESS_KEY": "someSecretAccessKey",
	}
	cfg, err := Load(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AWSRegion != "eu-west-1" {
		t.Fatalf("expected AWSRegion eu-west-1, got %q", cfg.AWSRegion)
	}
	if cfg.Concurrency != 8 {
		t.Fatalf("expected Concurrency 8, got %d", cfg.Concurrency)
	}
	if cfg.SQSEndpoint != "http://localhost:9324" {
		t.Fatalf("expected SQSEndpoint http://localhost:9324, got %q", cfg.SQSEndpoint)
	}
	if cfg.AWSSecretKey != "someSecretAccessKey" {
		t.Fatalf("expected AWSSecretKey someSecretAccessKey, got %q", cfg.AWSSecretKey)
	}
	if cfg.AWSAccessKey != "someAccessKey" {
		t.Fatalf("expected AWSAccessKey someAccessKey, got %q", cfg.AWSAccessKey)
	}
}

func TestLoad_InvalidConcurrencyFails(t *testing.T) {
	env := fakeEnv{
		"SQS_QUEUE_URL":      "http://example.com/queue",
		"WORKER_CONCURRENCY": "0",
	}
	_, err := Load(env)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_NonIntegerConcurrencyFails(t *testing.T) {
	env := fakeEnv{
		"SQS_QUEUE_URL":      "http://example.com/queue",
		"WORKER_CONCURRENCY": "abc",
	}
	_, err := Load(env)
	if err == nil {
		t.Fatal("expected error for non-integer WORKER_CONCURRENCY, got nil")
	}
}

func TestLoad_InvalidMaxInFlightFails(t *testing.T) {
	env := fakeEnv{
		"SQS_QUEUE_URL": "http://example.com/queue",
		"MAX_IN_FLIGHT": "0",
	}
	_, err := Load(env)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_NonIntegerMaxInFlightFails(t *testing.T) {
	env := fakeEnv{
		"SQS_QUEUE_URL": "http://example.com/queue",
		"MAX_IN_FLIGHT": "abc",
	}
	_, err := Load(env)
	if err == nil {
		t.Fatal("expected error for non-integer WORKER_CONCURRENCY, got nil")
	}
}
