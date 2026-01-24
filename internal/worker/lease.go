package worker

import (
	"context"
	"time"
)

type LeaseStore interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error)
	Release(ctx context.Context, key string, token string) error
}
