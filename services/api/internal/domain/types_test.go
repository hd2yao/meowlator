package domain

import "testing"

func TestParseIntentLabel(t *testing.T) {
	if _, err := ParseIntentLabel("FEEDING"); err != nil {
		t.Fatalf("expected valid label, got error: %v", err)
	}
	if _, err := ParseIntentLabel("NOT_A_LABEL"); err == nil {
		t.Fatal("expected invalid label error")
	}
}

func TestInferenceResultValidate(t *testing.T) {
	res := InferenceResult{
		IntentTop3: []IntentProb{{Label: IntentFeeding, Prob: 0.8}},
		State:      State3D{Tension: LevelLow, Arousal: LevelMid, Comfort: LevelHigh},
		Confidence: 0.8,
		Source:     "EDGE",
	}
	if err := res.Validate(); err != nil {
		t.Fatalf("expected valid inference result, got %v", err)
	}
}
