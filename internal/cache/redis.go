package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const ttl = 24 * time.Hour

type Cache struct {
	client *redis.Client
}

func New(host string, port int) *Cache {
	return &Cache{
		client: redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d", host, port),
		}),
	}
}

func (c *Cache) Get(ctx context.Context, code string) (string, error) {
	val, err := c.client.Get(ctx, code).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (c *Cache) Set(ctx context.Context, code, originalURL string) error {
	return c.client.Set(ctx, code, originalURL, ttl).Err()
}
