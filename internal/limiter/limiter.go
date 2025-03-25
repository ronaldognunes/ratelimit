package limiter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ronaldognunes/ratelimiter/internal/database"
)

type Limiter struct {
	store            database.RateLimitStore
	limitPerSecIp    int
	limitPerSecToken int
	blockDuration    time.Duration
	mu               sync.Mutex
	blockedKeys      map[string]time.Time
}

func NewLimiter(store database.RateLimitStore, limitPerSecIp, limitPerSecToken, blockDurationSec int) *Limiter {
	return &Limiter{
		store:            store,
		limitPerSecIp:    limitPerSecIp,
		limitPerSecToken: limitPerSecToken,
		blockDuration:    time.Duration(blockDurationSec) * time.Second,
		blockedKeys:      make(map[string]time.Time),
	}
}

func (l *Limiter) AllowRequest(ctx context.Context, typeLimit, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if unblockTime, exists := l.blockedKeys[key]; exists {
		if time.Now().Before(unblockTime) {
			return errors.New("you have reached the maximum number of requests or actions allowed within a certain time frame")
		}
		l.store.ResetRequest(ctx, key)
		delete(l.blockedKeys, key)
	}

	count, err := l.store.IncrementRequest(ctx, key)
	if err != nil {
		return err
	}

	if (typeLimit == "IP" && count > l.limitPerSecIp) || typeLimit == "TOKEN" && count > l.limitPerSecToken {
		l.blockedKeys[key] = time.Now().Add(l.blockDuration)
		return errors.New("you have reached the maximum number of requests or actions allowed within a certain time frame")
	}

	return nil
}
