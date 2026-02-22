package app

import (
	"strings"
	"testing"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

func TestParseCopyJSON(t *testing.T) {
	raw := `{"catLine":"喵","evidence":"因为耳朵后压","shareTitle":"主子发话"}`
	copy, err := ParseCopyJSON(raw)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if copy.CatLine == "" || copy.Evidence == "" || copy.ShareTitle == "" {
		t.Fatal("parsed copy fields should not be empty")
	}
}

func TestParseCopyJSONInvalid(t *testing.T) {
	if _, err := ParseCopyJSON(`{"catLine":"only one field"}`); err == nil {
		t.Fatal("expected parse error for incomplete payload")
	}
}

func TestEnforceRiskDisclaimer(t *testing.T) {
	input := domain.CopyBlock{
		CatLine:    "喵",
		Evidence:   "耳朵后压",
		ShareTitle: "主子发话",
	}
	result := domain.InferenceResult{
		Risk: &domain.RiskInfo{
			PainRiskScore: 0.8,
			PainRiskLevel: domain.PainRiskHigh,
			RiskEvidence:  []string{"test"},
			Disclaimer:    domain.PainRiskDisclaimer,
		},
	}
	output := enforceRiskDisclaimer(input, result)
	if !strings.Contains(output.Evidence, domain.PainRiskDisclaimer) {
		t.Fatalf("expected evidence to contain risk disclaimer")
	}
}
