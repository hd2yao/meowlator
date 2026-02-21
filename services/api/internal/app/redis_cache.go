package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type RedisCopyCache struct {
	client *redis.Client
}

func NewRedisCopyCache(addr string) (*RedisCopyCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisCopyCache{client: client}, nil
}

func (c *RedisCopyCache) Get(ctx context.Context, key string) (domain.CopyBlock, bool) {
	raw, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return domain.CopyBlock{}, false
	}
	var block domain.CopyBlock
	if err := json.Unmarshal([]byte(raw), &block); err != nil {
		return domain.CopyBlock{}, false
	}
	return block, true
}

func (c *RedisCopyCache) Set(ctx context.Context, key string, value domain.CopyBlock, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, key, raw, ttl).Err()
}
