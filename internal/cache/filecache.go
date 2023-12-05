package cache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/swmh/gopetbin/pkg/retry"
)

type ToReadCloser struct {
	io.Reader
}

func (t ToReadCloser) Close() error { return nil }

type FileCacheRedis struct {
	client *redis.Client
}

func NewFileCache(addr, username, password string, db int) (*FileCacheRedis, error) {
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
		return nil, fmt.Errorf("cannot connect to cache: %w", err)
	}

	return &FileCacheRedis{
		client: client,
	}, nil
}

func (c *FileCacheRedis) IsNoSuchPaste(err error) bool {
	return errors.Is(err, redis.Nil)
}

func (c *FileCacheRedis) Set(ctx context.Context, key string, value []byte) error {
	return c.client.Set(ctx, key, value, time.Duration(0)).Err()
}

func (c *FileCacheRedis) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	v, err := c.client.Get(ctx, key).Result()
	return ToReadCloser{bytes.NewReader([]byte(v))}, err
}
