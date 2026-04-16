package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dysania/meowlator/services/inference/internal/app"
)

type failingPredictor struct{}

func (f failingPredictor) Predict(imageKey string, sceneTag string) (app.InferenceResult, error) {
	return app.InferenceResult{}, errors.New("predict failed")
}

func (f failingPredictor) Name() string { return "failing" }

func TestPredictReturnsInternalServerErrorWhenPredictorFails(t *testing.T) {
	h := NewHandler(failingPredictor{})
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/inference/predict", strings.NewReader(`{"imageKey":"samples/u1/s1.jpg","sceneTag":"UNKNOWN"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "predict failed") {
		t.Fatalf("expected upstream error message, got %s", rec.Body.String())
	}
}
