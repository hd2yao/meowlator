package domain

import "testing"

func TestFeedbackWeight(t *testing.T) {
	if got := FeedbackWeight(true); got != WeightConfirmed {
		t.Fatalf("expected confirmed weight %.1f, got %.1f", WeightConfirmed, got)
	}
	if got := FeedbackWeight(false); got != WeightCorrected {
		t.Fatalf("expected corrected weight %.1f, got %.1f", WeightCorrected, got)
	}
}

func TestReliabilityScore(t *testing.T) {
	if got := ReliabilityScore(50, 0.2, false); got <= 0.9 {
		t.Fatalf("expected boosted score, got %.2f", got)
	}
	if got := ReliabilityScore(3, 0.9, true); got >= 0.5 {
		t.Fatalf("expected penalized score, got %.2f", got)
	}
}
