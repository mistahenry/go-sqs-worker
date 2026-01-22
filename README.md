# go-sqs-worker

A Go-based Amazon SQS worker project focused on building a production-style
message consumer, with deterministic local development using ElasticMQ.

This repository is intentionally incremental. It starts with a solid local SQS
environment and smoke testing, and layers in production-grade worker behavior
step by step.

---

## Goals

- Provide a clean foundation for implementing a production-style SQS worker in Go
- Support fast local development without requiring an AWS account
- Make message consumption behavior explicit, testable, and evolvable

This project is not a framework. It is a reference-style implementation meant to
demonstrate correct patterns and tradeoffs.

---

## Local Development Setup

### Prerequisites

- Go (1.21+ recommended)
- Docker + Docker Compose
- AWS CLI (for local smoke testing)

No AWS credentials are required for local development.

---

### 1. Initialize environment variables

Create a local `.env` file from the example:

```
cp .env.example .env
```

The `.env` file is intentionally not committed. It is loaded automatically by the
Makefile and provides configuration for local SQS access.

---

### 2. Start local SQS (ElasticMQ)

This project uses ElasticMQ, an SQS-compatible message queue, for local
development and integration testing.

Start it with:

```
make dev-up
```

ElasticMQ will be available at:

```
http://localhost:9324
```

---

### 3. Verify send / receive with a smoke test

Run:

```
make sqs-smoke
```

This will:
- Ensure the queue exists
- Send a test message
- Receive and print the message body

If this succeeds, the local SQS environment is working correctly.

---

### 4. View logs or shut down

```
make dev-logs
make dev-down
```

---

## Configuration

Configuration is provided via environment variables.

Key variables:

- `SQS_ENDPOINT` – SQS API endpoint (local ElasticMQ or AWS)
- `SQS_QUEUE_URL` – Queue URL
- `SQS_QUEUE_NAME` – Queue name
- `AWS_REGION` – AWS region (required by AWS SDK and CLI)

All configuration is injected externally. The application does not load `.env`
files itself.

## Roadmap / TODO

Planned implementation steps, in order:

- [X] Scaffold Go worker entrypoint (`cmd/worker`)
- [X] Implement long-polling receive loop
- [X] Add bounded concurrency and backpressure
- [X] Delete messages only after successful processing
- [X] Graceful shutdown with in-flight drain
  - [ ] Test drain behavior with Unit Test 
- [ ] Idempotent handler interface (external coordination)
- [ ] Visibility timeout extension for long-running jobs
- [X] Unit tests with fake SQS client
- [X] Integration tests against ElasticMQ
- [ ] Implement Configurable logger pattern
- [ ] Optional: batch delete and metrics hooks

Each step is intended to be implemented and validated independently.

---

## ElasticMQ

Local SQS functionality is provided by ElasticMQ:

https://github.com/softwaremill/elasticmq

ElasticMQ is an SQS-compatible message queue that supports long polling,
visibility timeouts, and redelivery semantics, making it well-suited for local
development and testing of SQS consumers.

---

## Notes

- This project assumes **at-least-once delivery semantics**
- Idempotency is handled at the handler / business-logic layer, not by SQS
- Configuration is environment-driven and injected externally

---

## License

MIT
