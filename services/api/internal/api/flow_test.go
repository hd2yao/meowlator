package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dysania/meowlator/services/api/internal/app"
	"github.com/dysania/meowlator/services/api/internal/domain"
	"github.com/dysania/meowlator/services/api/internal/repository"
)

type flowInferenceClient struct {
	result domain.InferenceResult
}

func (f flowInferenceClient) Predict(ctx context.Context, imageKey string, sceneTag string) (*domain.InferenceResult, error) {
	_ = ctx
	_ = imageKey
	_ = sceneTag
	result := f.result
	return &result, nil
}

type flowCopyClient struct{}

func (f flowCopyClient) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	_ = ctx
	_ = result
	_ = styleVersion
	return domain.CopyBlock{
		CatLine:    "喵喵委员会投票 80%：本猫当前诉求是 WANT_PLAY，速速响应。",
		Evidence:   "依据：状态(紧张MID/兴奋MID/舒适LOW) + 视觉证据 观察到猫脸与躯干主体区域。",
		ShareTitle: "主子发话：WANT_PLAY（翻译版）",
	}, nil
}

func TestFlowLoginUploadFinalizeFeedback(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := app.NewService(
		repo,
		flowInferenceClient{
			result: domain.InferenceResult{
				IntentTop3: []domain.IntentProb{{Label: domain.IntentWantPlay, Prob: 0.82}},
				State:      domain.State3D{Tension: domain.LevelMid, Arousal: domain.LevelHigh, Comfort: domain.LevelLow},
				Confidence: 0.82,
				Source:     "CLOUD",
				Evidence:   []string{"cloud evidence"},
			},
		},
		flowCopyClient{},
		app.Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45},
		7,
		"model-v1",
		false,
		nil,
	)
	handler := NewHandler(svc, HandlerOptions{
		RateLimitPerUserMin: 120,
		RateLimitPerIPMin:   300,
		AdminToken:          "dev-admin-token",
		WhitelistEnabled:    false,
		WhitelistDailyQuota: 100,
	})
	mux := http.NewServeMux()
	handler.Register(mux)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	loginResp := mustDoJSON[struct {
		UserID       string `json:"userId"`
		SessionToken string `json:"sessionToken"`
		ExpiresAt    int64  `json:"expiresAt"`
	}](t, server.Client(), http.MethodPost, server.URL+"/v1/auth/wechat/login", map[string]string{"Content-Type": "application/json"}, []byte(`{"code":"wx-code-flow"}`))

	authHeaders := map[string]string{
		"Authorization": "Bearer " + loginResp.SessionToken,
		"X-User-Id":     loginResp.UserID,
	}

	uploadReqBody := []byte(`{"catId":"cat-flow","suffix":".jpg"}`)
	uploadURLResp := mustDoSignedJSON[struct {
		SampleID          string `json:"sampleId"`
		ImageKey          string `json:"imageKey"`
		UploadURL         string `json:"uploadUrl"`
		ExpiresInSeconds  int    `json:"expiresInSeconds"`
		RetentionDeadline int64  `json:"retentionDeadline"`
	}](t, server.Client(), server.URL, "/v1/samples/upload-url", authHeaders, uploadReqBody)

	if uploadURLResp.SampleID == "" {
		t.Fatalf("expected sample id from upload-url")
	}
	if !strings.Contains(uploadURLResp.ImageKey, uploadURLResp.SampleID) {
		t.Fatalf("expected image key to include sample id, got %s", uploadURLResp.ImageKey)
	}
	if uploadURLResp.UploadURL == "" {
		t.Fatalf("expected upload url")
	}

	uploadPath := filepath.Join(os.TempDir(), "meowlator", "uploads", uploadURLResp.SampleID+".jpg")
	t.Cleanup(func() {
		_ = os.Remove(uploadPath)
	})
	performMultipartUpload(t, server.Client(), uploadURLResp.UploadURL, authHeaders, "file", "cat.jpg", []byte("fake-image-bytes"))

	finalizeResp := mustDoJSON[struct {
		SampleID string `json:"sampleId"`
		Result   struct {
			Source     string  `json:"source"`
			Confidence float64 `json:"confidence"`
			EdgeMeta   *struct {
				FallbackUsed bool `json:"fallbackUsed"`
			} `json:"edgeMeta,omitempty"`
		} `json:"result"`
		NeedFeedback bool `json:"needFeedback"`
		FallbackUsed bool `json:"fallbackUsed"`
	}](t, server.Client(), http.MethodPost, server.URL+"/v1/inference/finalize", authHeaders, mustJSONBody(map[string]any{
		"sampleId":      uploadURLResp.SampleID,
		"deviceCapable": true,
		"sceneTag":      "PLAY",
		"edgeResult": map[string]any{
			"intentTop3": []map[string]any{
				{"label": "WANT_PLAY", "prob": 0.55},
				{"label": "FEEDING", "prob": 0.25},
				{"label": "CURIOUS_OBSERVE", "prob": 0.20},
			},
			"state": map[string]any{
				"tension": "MID",
				"arousal": "MID",
				"comfort": "LOW",
			},
			"confidence":       0.55,
			"source":           "EDGE",
			"evidence":         []string{"edge evidence"},
			"copyStyleVersion": "v1",
		},
		"edgeRuntime": map[string]any{
			"engine":        "wx-heuristic-v1",
			"modelVersion":  "mobilenetv3-small-int8-v2",
			"modelHash":     "dev-hash-v1",
			"inputShape":    "1x3x224x224",
			"loadMs":        12,
			"inferMs":       38,
			"deviceModel":   "iPhone15,2",
			"failureCode":   "EDGE_RUNTIME_ERROR",
			"failureReason": "",
		},
	}))

	if finalizeResp.SampleID != uploadURLResp.SampleID {
		t.Fatalf("expected same sample id, got %s", finalizeResp.SampleID)
	}
	if finalizeResp.Result.Source != "CLOUD" {
		t.Fatalf("expected cloud fallback result, got %s", finalizeResp.Result.Source)
	}
	if finalizeResp.Result.Confidence < 0.8 {
		t.Fatalf("expected confidence from fake cloud result, got %.2f", finalizeResp.Result.Confidence)
	}
	if !finalizeResp.FallbackUsed {
		t.Fatalf("expected fallbackUsed=true")
	}
	if finalizeResp.NeedFeedback {
		t.Fatalf("expected needFeedback=false for middle-confidence edge fallback")
	}

	feedbackResp := mustDoJSON[domain.Feedback](t, server.Client(), http.MethodPost, server.URL+"/v1/feedback", authHeaders, mustJSONBody(map[string]any{
		"sampleId":  uploadURLResp.SampleID,
		"isCorrect": false,
		"trueLabel": "WANT_PLAY",
	}))
	if feedbackResp.SampleID != uploadURLResp.SampleID {
		t.Fatalf("expected feedback sample id %s, got %s", uploadURLResp.SampleID, feedbackResp.SampleID)
	}
	if feedbackResp.TrueLabel != domain.IntentWantPlay {
		t.Fatalf("expected true label WANT_PLAY, got %s", feedbackResp.TrueLabel)
	}
	if feedbackResp.TrainingWeight != domain.WeightCorrected {
		t.Fatalf("expected corrected weight, got %.1f", feedbackResp.TrainingWeight)
	}
}

func mustDoSignedJSON[T any](t *testing.T, client *http.Client, baseURL string, path string, authHeaders map[string]string, body []byte) T {
	t.Helper()
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := computeRequestSignature(http.MethodPost, path, ts, string(body), strings.TrimPrefix(authHeaders["Authorization"], "Bearer "))
	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeaders["Authorization"])
	req.Header.Set("X-User-Id", authHeaders["X-User-Id"])
	req.Header.Set("X-Req-Ts", ts)
	req.Header.Set("X-Req-Sig", sig)
	return mustDoResponse[T](t, client, req)
}

func mustDoJSON[T any](t *testing.T, client *http.Client, method string, url string, headers map[string]string, body []byte) T {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	return mustDoResponse[T](t, client, req)
}

func mustDoResponse[T any](t *testing.T, client *http.Client, req *http.Request) T {
	t.Helper()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", req.Method, req.URL.String(), err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("request %s %s returned %d: %s", req.Method, req.URL.String(), resp.StatusCode, string(raw))
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, string(raw))
	}
	return out
}

func performMultipartUpload(t *testing.T, client *http.Client, uploadURL string, headers map[string]string, fieldName string, fileName string, data []byte) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, uploadURL, &buf)
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("upload returned %d: %s", resp.StatusCode, string(raw))
	}
}

func mustJSONBody(payload any) []byte {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return raw
}
