package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryLeaseStore is a test-only implementation.
// It does not enforce TTL; use Expire to simulate lease expiry.
type MemoryLeaseStore struct {
	mu     sync.Mutex
	leases map[string]string
}

func NewMemoryLeaseStore() *MemoryLeaseStore {
	return &MemoryLeaseStore{
		leases: make(map[string]string),
	}
}

func (m *MemoryLeaseStore) Acquire(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	if ttl <= 0 {
		return "", false, fmt.Errorf("lease ttl must be > 0")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.leases[key]; exists {
		return "", false, nil
	}

	token := uuid.New().String()
	m.leases[key] = token
	return token, true, nil
}

func (m *MemoryLeaseStore) Release(ctx context.Context, key string, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.leases[key] == token {
		delete(m.leases, key)
	}
	return nil
}

// Expire forces a lease to expire. Test use only.
func (m *MemoryLeaseStore) Expire(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.leases, key)
}
