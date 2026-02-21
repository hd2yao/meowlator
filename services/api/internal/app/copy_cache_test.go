package app

import (
	"context"
	"testing"
	"time"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type memoryCopyCache struct {
	store map[string]domain.CopyBlock
}

func (m *memoryCopyCache) Get(ctx context.Context, key string) (domain.CopyBlock, bool) {
	_ = ctx
	v, ok := m.store[key]
	return v, ok
}

func (m *memoryCopyCache) Set(ctx context.Context, key string, value domain.CopyBlock, ttl time.Duration) {
	_ = ctx
	_ = ttl
	m.store[key] = value
}

type countCopyClient struct {
	calls int
}

func (c *countCopyClient) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	_ = ctx
	_ = result
	_ = styleVersion
	c.calls++
	return domain.CopyBlock{CatLine: "喵", Evidence: "依据", ShareTitle: "标题"}, nil
}

func TestCachingCopyClient(t *testing.T) {
	base := &countCopyClient{}
	cache := &memoryCopyCache{store: make(map[string]domain.CopyBlock)}
	client := NewCachingCopyClient(base, cache, time.Hour)

	result := domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentFeeding, Prob: 0.7}},
		State:      domain.State3D{Tension: domain.LevelLow, Arousal: domain.LevelMid, Comfort: domain.LevelHigh},
		Confidence: 0.7,
		Source:     "EDGE",
	}
	_, err := client.Generate(context.Background(), result, "v1")
	if err != nil {
		t.Fatalf("first generate failed: %v", err)
	}
	_, err = client.Generate(context.Background(), result, "v1")
	if err != nil {
		t.Fatalf("second generate failed: %v", err)
	}
	if base.calls != 1 {
		t.Fatalf("expected 1 underlying call, got %d", base.calls)
	}
}
