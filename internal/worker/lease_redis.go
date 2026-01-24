package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const leaseKeyPrefix = "lease:"

var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

type RedisLeaseStore struct {
	client *redis.Client
}

func NewRedisLeaseStore(client *redis.Client) *RedisLeaseStore {
	return &RedisLeaseStore{client: client}
}

func (r *RedisLeaseStore) Acquire(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	if ttl <= 0 {
		return "", false, fmt.Errorf("lease ttl must be > 0")
	}
	token := uuid.New().String()
	redisKey := leaseKeyPrefix + key
	ok, err := r.client.SetNX(ctx, redisKey, token, ttl).Result()
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	return token, true, nil
}

func (r *RedisLeaseStore) Release(ctx context.Context, key string, token string) error {
	redisKey := leaseKeyPrefix + key
	_, err := releaseScript.Run(ctx, r.client, []string{redisKey}, token).Result()
	return err
}
