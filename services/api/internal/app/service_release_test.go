package app

import (
	"context"
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

	if err := svc.RegisterModelEvaluation(context.Background(), "model-v2", `{"top1":0.63}`); err != nil {
		t.Fatalf("register model failed: %v", err)
	}
	if err := svc.RolloutModel(context.Background(), "model-v2", 0.3, 1); err != nil {
		t.Fatalf("rollout model failed: %v", err)
	}
	cfg := svc.ClientConfig("u1")
	if cfg.ModelRollout.ActiveModel != "model-v2" {
		t.Fatalf("expected model-v2 in rollout, got %s", cfg.ModelRollout.ActiveModel)
	}
	if err := svc.ActivateModel(context.Background(), "model-v2"); err != nil {
		t.Fatalf("activate model failed: %v", err)
	}
	cfg = svc.ClientConfig("u1")
	if cfg.ModelRollout.RolloutRatio != 1.0 {
		t.Fatalf("expected rollout ratio=1, got %f", cfg.ModelRollout.RolloutRatio)
	}
}
