package database

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RateLimitStore interface {
	IncrementRequest(ctx context.Context, key string) (int, error)
	ResetRequest(ctx context.Context, key string) error
}

type RedisStore struct {
	Client *redis.Client
}

func NewRedisStore(addr string) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisStore{
		Client: client,
	}
}

func (r *RedisStore) IncrementRequest(ctx context.Context, key string) (int, error) {
	count, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *RedisStore) ResetRequest(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}
