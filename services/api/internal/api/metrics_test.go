package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dysania/meowlator/services/api/internal/app"
	"github.com/dysania/meowlator/services/api/internal/domain"
	"github.com/dysania/meowlator/services/api/internal/repository"
)

type metricsInferenceClient struct{}

func (m metricsInferenceClient) Predict(ctx context.Context, imageKey string, sceneTag string) (*domain.InferenceResult, error) {
	_ = ctx
	_ = imageKey
	_ = sceneTag
	result := domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentWantPlay, Prob: 0.82}},
		State:      domain.State3D{Tension: domain.LevelMid, Arousal: domain.LevelHigh, Comfort: domain.LevelLow},
		Confidence: 0.82,
		Source:     "CLOUD",
		Evidence:   []string{"cloud evidence"},
	}
	return &result, nil
}

type metricsCopyClient struct{}

func (m metricsCopyClient) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	_ = ctx
	_ = result
	_ = styleVersion
	return domain.CopyBlock{CatLine: "喵", Evidence: "依据", ShareTitle: "标题"}, nil
}

func TestMetricsEndpointIncludesFinalizeAndCopyCounters(t *testing.T) {
	metrics := NewMetrics()
	repo := repository.NewMemoryRepository()
	copyClient := app.NewObservedCopyClient(metricsCopyClient{}, metrics)
	svc := app.NewService(
		repo,
		metricsInferenceClient{},
		copyClient,
		app.Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45},
		7,
		"model-v1",
		false,
		nil,
	)

	upload, err := svc.CreateUploadSample(context.Background(), app.CreateUploadSampleInput{UserID: "u1", CatID: "c1"})
	if err != nil {
		t.Fatalf("create sample failed: %v", err)
	}
	login, err := svc.Login(context.Background(), app.LoginInput{Code: "wx-code-metrics"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	handler := NewHandler(svc, HandlerOptions{
		RateLimitPerUserMin: 120,
		RateLimitPerIPMin:   300,
		WhitelistDailyQuota: 100,
		Metrics:             metrics,
	})
	mux := http.NewServeMux()
	handler.Register(mux)

	body := strings.NewReader(`{"sampleId":"` + upload.SampleID + `","deviceCapable":false,"sceneTag":"PLAY"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/inference/finalize", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+login.SessionToken)
	req.Header.Set("X-User-Id", login.UserID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected finalize status %d body=%s", rec.Code, rec.Body.String())
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	mux.ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("expected metrics endpoint 200, got %d", metricsRec.Code)
	}
	raw := metricsRec.Body.String()
	for _, wanted := range []string{
		"api_requests_total 1",
		"api_errors_total 0",
		"finalize_requests_total 1",
		"finalize_fallback_total 1",
		"finalize_duration_ms_count 1",
		"copy_requests_total 1",
	} {
		if !strings.Contains(raw, wanted) {
			t.Fatalf("expected metrics output to contain %s, got %s", wanted, raw)
		}
	}
}
