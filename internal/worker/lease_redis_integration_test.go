//go:build integration

package worker_test

import (
	"context"
	"testing"
	"time"

	"go-sqs-worker/internal/worker"

	"github.com/redis/go-redis/v9"
)

func TestRedisLeaseStore_AcquireRelease(t *testing.T) {
	client, ctx := newTestRedis(t)
	store := worker.NewRedisLeaseStore(client)

	// First acquire succeeds
	token1, ok, err := store.Acquire(ctx, "acquire-release-1", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected to acquire lease")
	}
	if token1 == "" {
		t.Fatal("expected non-empty token")
	}

	// Second acquire fails
	token2, ok, err := store.Acquire(ctx, "acquire-release-1", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected lease to be held")
	}
	if token2 != "" {
		t.Fatal("expected empty token on failed acquire")
	}

	// Release with correct token
	if err := store.Release(ctx, "acquire-release-1", token1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now acquire succeeds again
	token3, ok, err := store.Acquire(ctx, "acquire-release-1", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected to acquire after release")
	}
	if token3 == "" {
		t.Fatal("expected non-empty token")
	}
	if token1 == token3 {
		t.Fatal("expected different token on next acquire after release")
	}
}

func TestRedisLeaseStore_TTLExpiry(t *testing.T) {
	client, ctx := newTestRedis(t)
	store := worker.NewRedisLeaseStore(client)

	// Acquire with short TTL
	token1, ok, err := store.Acquire(ctx, "ttl-expiry-1", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected to acquire lease")
	}

	// Immediately fails
	_, ok, err = store.Acquire(ctx, "ttl-expiry-1", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected lease to be held")
	}

	// Wait for TTL + buffer
	time.Sleep(400 * time.Millisecond)

	// Now succeeds with new token
	token2, ok, err := store.Acquire(ctx, "ttl-expiry-1", time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected to acquire after TTL expiry")
	}
	if token1 == token2 {
		t.Fatal("expected different token after expiry")
	}
}

func TestRedisLeaseStore_WrongTokenRelease(t *testing.T) {
	client, ctx := newTestRedis(t)
	store := worker.NewRedisLeaseStore(client)

	// Acquire
	_, ok, err := store.Acquire(ctx, "wrong-token-1", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected to acquire lease")
	}

	// Release with wrong token does nothing
	if err := store.Release(ctx, "wrong-token-1", "wrong-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Still held
	token2, ok, err := store.Acquire(ctx, "wrong-token-1", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected lease to still be held")
	}
	if token2 != "" {
		t.Fatal("expected empty token on failed acquire")
	}
}

func newTestRedis(t *testing.T) (*redis.Client, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	c := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = c.Close() })

	if err := c.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unreachable: %v", err)
	}
	// safe for now but potentially heavy
	if err := c.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flushdb failed: %v", err)
	}
	return c, ctx
}
