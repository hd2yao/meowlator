package app

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPredictDeterministic(t *testing.T) {
	m := NewModel(nil)
	one, err := m.Predict("samples/u1/s1.jpg", "FOOD_BOWL")
	if err != nil {
		t.Fatalf("predict one failed: %v", err)
	}
	two, err := m.Predict("samples/u1/s1.jpg", "FOOD_BOWL")
	if err != nil {
		t.Fatalf("predict two failed: %v", err)
	}
	if one.IntentTop3[0].Label != two.IntentTop3[0].Label || one.Confidence != two.Confidence {
		t.Fatalf("expected deterministic result")
	}
	if len(one.IntentTop3) != 3 {
		t.Fatalf("expected top3 intents, got %d", len(one.IntentTop3))
	}
	if one.Source != "CLOUD" {
		t.Fatalf("expected cloud source, got %s", one.Source)
	}
}

func TestLoadIntentPriors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "intent_priors.json")
	raw := `{"intent_priors":{"FEEDING":2.0,"WANT_PLAY":1.0}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write priors file failed: %v", err)
	}

	priors, err := LoadIntentPriors(path)
	if err != nil {
		t.Fatalf("load priors failed: %v", err)
	}
	if len(priors) != 2 {
		t.Fatalf("expected 2 priors, got %d", len(priors))
	}
	total := 0.0
	for _, value := range priors {
		total += value
	}
	if math.Abs(total-1.0) > 0.0001 {
		t.Fatalf("expected normalized priors sum=1.0, got %.6f", total)
	}
}

func TestPredictWithPriorsAddsEvidence(t *testing.T) {
	m := NewModel(map[IntentLabel]float64{IntentFeeding: 1.0})
	res, err := m.Predict("samples/u1/s1.jpg", "FOOD_BOWL")
	if err != nil {
		t.Fatalf("predict with priors failed: %v", err)
	}
	if len(res.Evidence) < 3 {
		t.Fatalf("expected priors evidence appended")
	}
	joined := strings.Join(res.Evidence, ",")
	if !strings.Contains(joined, "先验") {
		t.Fatalf("expected priors keyword in evidence, got %s", joined)
	}
}
