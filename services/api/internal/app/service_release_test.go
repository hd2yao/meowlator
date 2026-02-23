package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/dysania/meowlator/services/api/internal/repository"
)

func TestLoginAndValidateSession(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewService(repo, fakeInference{}, fakeCopy{}, Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45}, 7, "model-v1", false, nil)

	out, err := svc.Login(context.Background(), LoginInput{Code: "wx-code-1"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if out.SessionToken == "" || out.UserID == "" {
		t.Fatalf("expected session token and user id")
	}
	if err := svc.ValidateSession(context.Background(), out.UserID, out.SessionToken); err != nil {
		t.Fatalf("validate session failed: %v", err)
	}
}

func TestRolloutAndActivateModel(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewService(repo, fakeInference{}, fakeCopy{}, Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45}, 7, "model-v1", false, nil)

	if err := svc.RegisterModelEvaluation(context.Background(), "model-v1", `{"top1":0.61}`); err != nil {
		t.Fatalf("register baseline model failed: %v", err)
	}
	if err := svc.ActivateModel(context.Background(), "model-v1"); err != nil {
		t.Fatalf("activate baseline model failed: %v", err)
	}
	if err := svc.RegisterModelEvaluation(context.Background(), "model-v2", `{"top1":0.63}`); err != nil {
		t.Fatalf("register model failed: %v", err)
	}
	inRolloutUser := "u-in-rollout"
	targetBucket := repository.ABBucket(inRolloutUser, 100)
	if err := svc.RolloutModel(context.Background(), "model-v2", 0.01, targetBucket); err != nil {
		t.Fatalf("rollout model failed: %v", err)
	}

	cfg := svc.ClientConfig(inRolloutUser)
	if cfg.ModelRollout.ActiveModel != "model-v1" {
		t.Fatalf("expected active baseline model-v1, got %s", cfg.ModelRollout.ActiveModel)
	}
	if cfg.ModelRollout.RolloutModel != "model-v2" {
		t.Fatalf("expected rollout model-v2, got %s", cfg.ModelRollout.RolloutModel)
	}
	if !cfg.ModelRollout.InRollout {
		t.Fatalf("expected inRollout=true for user %s", inRolloutUser)
	}
	if cfg.ModelVersion != "model-v2" {
		t.Fatalf("expected selected model-v2 for in-rollout user, got %s", cfg.ModelVersion)
	}

	outsideUser := findUserDifferentBucket(targetBucket)
	cfgOutside := svc.ClientConfig(outsideUser)
	if cfgOutside.ModelVersion != "model-v1" {
		t.Fatalf("expected baseline model-v1 for outside user, got %s", cfgOutside.ModelVersion)
	}
	if cfgOutside.ModelRollout.InRollout {
		t.Fatalf("expected inRollout=false for outside user")
	}

	if err := svc.ActivateModel(context.Background(), "model-v2"); err != nil {
		t.Fatalf("activate model failed: %v", err)
	}
	cfg = svc.ClientConfig("u1")
	if cfg.ModelVersion != "model-v2" {
		t.Fatalf("expected active model-v2 after activation, got %s", cfg.ModelVersion)
	}
}

func findUserDifferentBucket(targetBucket int) string {
	for i := 0; i < 1000; i++ {
		candidate := fmt.Sprintf("u-outside-%d", i)
		if repository.ABBucket(candidate, 100) != targetBucket {
			return candidate
		}
	}
	return "u-outside-fallback"
}

func TestHitRolloutBucketWrapAround(t *testing.T) {
	if !hitRolloutBucket(0, 95, 100, 0.1) {
		t.Fatalf("expected bucket 0 to hit wrap-around rollout window from target 95")
	}
	if hitRolloutBucket(50, 95, 100, 0.1) {
		t.Fatalf("expected bucket 50 to miss rollout window from target 95")
	}
	if !hitRolloutBucket(50, 0, 100, 1.0) {
		t.Fatalf("ratio=1 should include all buckets")
	}
	if hitRolloutBucket(0, 0, 100, 0.0) {
		t.Fatalf("ratio=0 should not include any bucket")
	}
}
