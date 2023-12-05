package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/swmh/gopetbin/internal/service"
	"github.com/swmh/gopetbin/pkg/retry"
)

type Paste struct {
	Name       string    `json:"name"`
	Expire     time.Time `json:"expire"`
	BurnAfter  int       `json:"burn_after"`
	IsBurnable bool      `json:"is_burnable"`
}

func (p *Paste) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}

type CacheRedis struct {
	client *redis.Client
}

func New(addr, username, password string, db int) (*CacheRedis, error) {
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

	return &CacheRedis{
		client: client,
	}, nil
}

func (c *CacheRedis) IsNoSuchPaste(err error) bool {
	return errors.Is(err, redis.Nil)
}

func (c *CacheRedis) Unmarshal(_ context.Context, value string) (service.Paste, error) {
	var paste Paste
	err := json.Unmarshal([]byte(value), &paste)
	return service.Paste(paste), err
}

const errorValue = "error"

func (c *CacheRedis) IsError(_ context.Context, value string) bool {
	return value == errorValue
}

func (c *CacheRedis) SetError(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Set(ctx, key, errorValue, ttl).Err()
}

func (c *CacheRedis) Set(ctx context.Context, key string, value service.Paste) error {
	v, err := json.Marshal(Paste(value))
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, v, time.Duration(0)).Err()
}

func (c *CacheRedis) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}
