package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type CopyCache interface {
	Get(ctx context.Context, key string) (domain.CopyBlock, bool)
	Set(ctx context.Context, key string, value domain.CopyBlock, ttl time.Duration)
}

type CachingCopyClient struct {
	next  CopyClient
	cache CopyCache
	ttl   time.Duration
}

func NewCachingCopyClient(next CopyClient, cache CopyCache, ttl time.Duration) *CachingCopyClient {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &CachingCopyClient{next: next, cache: cache, ttl: ttl}
}

func (c *CachingCopyClient) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	key := copyCacheKey(result, styleVersion)
	if cached, ok := c.cache.Get(ctx, key); ok {
		return cached, nil
	}
	generated, err := c.next.Generate(ctx, result, styleVersion)
	if err != nil {
		return domain.CopyBlock{}, err
	}
	c.cache.Set(ctx, key, generated, c.ttl)
	return generated, nil
}

func copyCacheKey(result domain.InferenceResult, styleVersion string) string {
	canonical, _ := json.Marshal(struct {
		Top3  []domain.IntentProb `json:"top3"`
		State domain.State3D      `json:"state"`
		Style string              `json:"style"`
	}{Top3: result.IntentTop3, State: result.State, Style: styleVersion})
	sum := sha1.Sum(canonical)
	return "copy:v1:" + hex.EncodeToString(sum[:])
}
