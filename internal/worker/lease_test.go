package worker

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLeaseStore_Acquire_Success(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	token, ok, err := store.Acquire(ctx, "job-1", time.Minute)

	if err != nil {
		t.Fatalf("unexpected error acquiring lease: %v", err)
	}

	if !ok {
		t.Fatalf("expected ok")
	}

	if token == "" {
		t.Fatalf("expected non-empty token")
	}
}

func TestMemoryLeaseStore_Acquire_AlreadyHeld(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	_, _, err := store.Acquire(ctx, "job-1", time.Minute)

	if err != nil {
		t.Fatalf("unexpected error acquiring lease: %v", err)
	}

	token, ok, err := store.Acquire(ctx, "job-1", time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ok {
		t.Fatalf("expected lease acquire to fail")
	}

	if token != "" {
		t.Fatalf("expected empty token on acquire fail")
	}
}

func TestMemoryLeaseStore_Acquire_DifferentKeys(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	_, ok1, err1 := store.Acquire(ctx, "job-1", time.Minute)
	_, ok2, err2 := store.Acquire(ctx, "job-2", time.Minute)

	if err1 != nil {
		t.Fatalf("unexpected error acquiring lease: %v", err1)
	}

	if err2 != nil {
		t.Fatalf("unexpected error acquiring lease: %v", err2)
	}

	if !ok1 || !ok2 {
		t.Fatal("expected to acquire both leases")
	}
}

func TestMemoryLeaseStore_Release_Success(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	token, _, _ := store.Acquire(ctx, "job-1", time.Minute)

	err := store.Release(ctx, "job-1", token)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be able to acquire again
	token2, ok, _ := store.Acquire(ctx, "job-1", time.Minute)
	if !ok {
		t.Fatal("expected to acquire lease after release")
	}

	if token == token2 {
		t.Fatal("expected to a new token upon re-acquire for the same job key")
	}
}

func TestMemoryLeaseStore_Release_WrongToken(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	_, _, _ = store.Acquire(ctx, "job-1", time.Minute)

	// Release with wrong token should do nothing
	err := store.Release(ctx, "job-1", "wrong-token")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Lease should still be held
	_, ok, _ := store.Acquire(ctx, "job-1", time.Minute)
	if ok {
		t.Fatal("expected lease to still be held")
	}
}

func TestMemoryLeaseStore_Expire(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	token, _, _ := store.Acquire(ctx, "job-1", time.Minute)

	store.Expire("job-1")

	// Should be able to acquire again after expiry
	token2, ok, _ := store.Acquire(ctx, "job-1", time.Minute)
	if !ok {
		t.Fatal("expected to acquire lease after expiry")
	}

	if token == token2 {
		t.Fatal("expected to a new token upon re-acquire for the same job key")
	}
}
