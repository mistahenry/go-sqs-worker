# Makefile for local SQS (ElasticMQ) + smoke testing

# Load .env if present (do not fail if missing)
ifneq (,$(wildcard .env))
	include .env
	export
endif

SHELL := /bin/bash

.PHONY: help init-env check-env dev-up dev-down dev-logs sqs-create-queue sqs-smoke

help:
	@echo "Targets:"
	@echo "  init-env   Create .env from .env.example if missing"
	@echo "  check-env  Verify required env vars are set"
	@echo "  dev-up     Start ElasticMQ"
	@echo "  dev-down   Stop ElasticMQ (and remove volumes)"
	@echo "  dev-logs   Tail ElasticMQ logs"
	@echo "  sqs-smoke  Send + receive one message via AWS CLI"

init-env:
	@cp -n .env.example .env 2>/dev/null || true
	@echo "Ensured .env exists (created from .env.example if missing)."

check-env:
	@if [ -z "$(SQS_ENDPOINT)" ] || [ -z "$(SQS_QUEUE_URL)" ]; then \
		echo "Missing env. Ensure .env exists and defines SQS_ENDPOINT and SQS_QUEUE_URL."; \
		exit 1; \
	fi
	@echo "SQS_ENDPOINT=$(SQS_ENDPOINT)"
	@echo "SQS_QUEUE_URL=$(SQS_QUEUE_URL)"

dev-up:
	docker compose up -d

dev-down:
	docker compose down -v

dev-logs:
	docker compose logs -f elasticmq

sqs-create-queue: check-env
	@aws --endpoint-url $(SQS_ENDPOINT) sqs create-queue --queue-name local-sqs-worker >/dev/null 2>&1 || true

sqs-smoke: check-env sqs-create-queue
	@echo "Sending + receiving one message..."
	@AWS_PAGER="" aws --endpoint-url $(SQS_ENDPOINT) sqs send-message \
	  --queue-url $(SQS_QUEUE_URL) \
	  --message-body '{"id":"smoke","hello":"world"}' >/dev/null
	@AWS_PAGER="" aws --endpoint-url $(SQS_ENDPOINT) sqs receive-message \
	  --queue-url $(SQS_QUEUE_URL) \
	  --wait-time-seconds 1 \
	  --max-number-of-messages 1 \
	  --query 'Messages[0].Body' \
	  --output text