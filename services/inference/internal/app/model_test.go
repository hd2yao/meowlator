package app

import "testing"

func TestPredictDeterministic(t *testing.T) {
	m := NewModel()
	one := m.Predict("samples/u1/s1.jpg", "FOOD_BOWL")
	two := m.Predict("samples/u1/s1.jpg", "FOOD_BOWL")
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
