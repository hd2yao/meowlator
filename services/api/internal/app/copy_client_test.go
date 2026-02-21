package app

import "testing"

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
