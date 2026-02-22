package domain

import "testing"

func TestEvaluatePainRiskHigh(t *testing.T) {
	result := InferenceResult{
		IntentTop3: []IntentProb{{Label: IntentDefensiveAlert, Prob: 0.8}},
		State:      State3D{Tension: LevelHigh, Arousal: LevelHigh, Comfort: LevelLow},
		Confidence: 0.8,
		Source:     "CLOUD",
		Evidence:   []string{"耳朵后压"},
	}
	risk := EvaluatePainRisk(result)
	if risk.PainRiskLevel != PainRiskHigh {
		t.Fatalf("expected high risk, got %s", risk.PainRiskLevel)
	}
	if err := risk.Validate(); err != nil {
		t.Fatalf("expected valid risk: %v", err)
	}
}

func TestRiskInfoValidate(t *testing.T) {
	risk := &RiskInfo{
		PainRiskScore: 0.6,
		PainRiskLevel: PainRiskMid,
		RiskEvidence:  []string{"test"},
		Disclaimer:    PainRiskDisclaimer,
	}
	if err := risk.Validate(); err != nil {
		t.Fatalf("expected valid risk info: %v", err)
	}

	risk.Disclaimer = ""
	if err := risk.Validate(); err == nil {
		t.Fatal("expected empty disclaimer to be rejected")
	}
}
