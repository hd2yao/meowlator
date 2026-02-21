package domain

import "testing"

func TestDecideThreshold(t *testing.T) {
	cfg := ThresholdConfig{EdgeAccept: 0.7, CloudFallback: 0.45}

	decision := DecideThreshold(0.8, true, cfg)
	if !decision.UseEdge || decision.NeedCloud {
		t.Fatalf("expected edge path, got %+v", decision)
	}

	decision = DecideThreshold(0.5, true, cfg)
	if !decision.NeedCloud || decision.UseEdge {
		t.Fatalf("expected cloud fallback for mid confidence, got %+v", decision)
	}

	decision = DecideThreshold(0.3, true, cfg)
	if !decision.NeedCloud || !decision.ForceFeedback {
		t.Fatalf("expected cloud + force feedback for low confidence, got %+v", decision)
	}
}
