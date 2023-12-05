package redlock

import (
	"context"
	"fmt"
	"time"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
	"github.com/swmh/gopetbin/internal/service"
	"github.com/swmh/gopetbin/pkg/retry"
)

type Redlock struct {
	client *redis.Client
	locker *redislock.Client
}

func New(addr, username, password string, db int) (*Redlock, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := retry.Retry(ctx, func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to lock: %w", err)
	}

	locker := redislock.New(client)

	return &Redlock{
		client: client,
		locker: locker,
	}, nil
}

type mutex struct {
	*redislock.Lock
}

func (m *mutex) Unlock(ctx context.Context) error {
	return m.Release(ctx)
}

func (l *Redlock) Lock(ctx context.Context, id string) (service.Mutex, error) {
	opts := redislock.Options{
		RetryStrategy: redislock.ExponentialBackoff(50*time.Millisecond, 5*time.Second),
		Metadata:      "",
		Token:         "",
	}
	lock, err := l.locker.Obtain(ctx, id, 5*time.Second, &opts)
	if err != nil {
		return nil, err
	}

	return &mutex{lock}, nil
}
