package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dysania/meowlator/services/api/internal/app"
	"github.com/dysania/meowlator/services/api/internal/domain"
)

type ctxKey string

const (
	ctxUserIDKey       ctxKey = "user_id"
	ctxSessionTokenKey ctxKey = "session_token"
)

type HandlerOptions struct {
	RateLimitPerUserMin int
	RateLimitPerIPMin   int
	AdminToken          string
}

type Handler struct {
	svc              *app.Service
	limiter          *requestLimiter
	rateLimitUserMin int
	rateLimitIPMin   int
	adminToken       string
	signatureMaxSkew time.Duration
}

func NewHandler(svc *app.Service, opts HandlerOptions) *Handler {
	if opts.RateLimitPerUserMin <= 0 {
		opts.RateLimitPerUserMin = 120
	}
	if opts.RateLimitPerIPMin <= 0 {
		opts.RateLimitPerIPMin = 300
	}
	return &Handler{
		svc:              svc,
		limiter:          newRequestLimiter(),
		rateLimitUserMin: opts.RateLimitPerUserMin,
		rateLimitIPMin:   opts.RateLimitPerIPMin,
		adminToken:       opts.AdminToken,
		signatureMaxSkew: 5 * time.Minute,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.withObserved(h.healthz))
	mux.HandleFunc("POST /v1/auth/wechat/login", h.withObserved(h.login))

	mux.HandleFunc("POST /v1/samples/upload-url", h.withObserved(h.withAuth(h.withRateLimit(h.withRequestSignature(h.createUploadURL)))))
	mux.HandleFunc("POST /v1/samples/upload/", h.withObserved(h.withAuth(h.withRateLimit(h.acceptUpload))))
	mux.HandleFunc("POST /v1/inference/finalize", h.withObserved(h.withAuth(h.withRateLimit(h.finalizeInference))))
	mux.HandleFunc("POST /v1/feedback", h.withObserved(h.withAuth(h.withRateLimit(h.saveFeedback))))
	mux.HandleFunc("POST /v1/copy/generate", h.withObserved(h.withAuth(h.withRateLimit(h.generateCopy))))
	mux.HandleFunc("DELETE /v1/samples/", h.withObserved(h.withAuth(h.withRateLimit(h.withRequestSignature(h.deleteSample)))))
	mux.HandleFunc("GET /v1/metrics/client-config", h.withObserved(h.withAuth(h.withRateLimit(h.clientConfig))))

	mux.HandleFunc("POST /v1/admin/models/register", h.withObserved(h.withAdmin(h.registerModel)))
	mux.HandleFunc("POST /v1/admin/models/rollout", h.withObserved(h.withAdmin(h.rolloutModel)))
	mux.HandleFunc("POST /v1/admin/models/activate", h.withObserved(h.withAdmin(h.activateModel)))
}

func (h *Handler) withObserved(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(recorder, r)
		userID := userIDFromContext(r.Context())
		if userID == "" {
			userID = "anonymous"
		}
		log.Printf("method=%s path=%s status=%d user=%s duration_ms=%d", r.Method, r.URL.Path, recorder.statusCode, userID, time.Since(startedAt).Milliseconds())
	}
}

func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
		token := bearerToken(r.Header.Get("Authorization"))
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "missing X-User-Id")
			return
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		if err := h.svc.ValidateSession(r.Context(), userID, token); err != nil {
			writeDomainError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserIDKey, userID)
		ctx = context.WithValue(ctx, ctxSessionTokenKey, token)
		next(w, r.WithContext(ctx))
	}
}

func (h *Handler) withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.adminToken == "" {
			writeError(w, http.StatusUnauthorized, "admin token not configured")
			return
		}
		if r.Header.Get("X-Admin-Token") != h.adminToken {
			writeError(w, http.StatusUnauthorized, "invalid admin token")
			return
		}
		next(w, r)
	}
}

func (h *Handler) withRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := userIDFromContext(r.Context())
		if userID != "" {
			if !h.limiter.Allow("user:"+userID, h.rateLimitUserMin, time.Now()) {
				writeError(w, http.StatusTooManyRequests, "user rate limit exceeded")
				return
			}
		}
		ip := clientIP(r)
		if !h.limiter.Allow("ip:"+ip, h.rateLimitIPMin, time.Now()) {
			writeError(w, http.StatusTooManyRequests, "ip rate limit exceeded")
			return
		}
		next(w, r)
	}
}

func (h *Handler) withRequestSignature(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken := sessionTokenFromContext(r.Context())
		if sessionToken == "" {
			writeError(w, http.StatusUnauthorized, "missing session context")
			return
		}
		ts := strings.TrimSpace(r.Header.Get("X-Req-Ts"))
		sig := strings.TrimSpace(r.Header.Get("X-Req-Sig"))
		if ts == "" || sig == "" {
			writeError(w, http.StatusUnauthorized, "missing request signature")
			return
		}
		tsInt, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid signature timestamp")
			return
		}
		now := time.Now().Unix()
		if abs64(now-tsInt) > int64(h.signatureMaxSkew.Seconds()) {
			writeError(w, http.StatusUnauthorized, "signature timestamp skew")
			return
		}

		bodyRaw, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cannot read request body")
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(bodyRaw)))
		expected := computeRequestSignature(r.Method, r.URL.Path, ts, string(bodyRaw), sessionToken)
		if sig != expected {
			writeError(w, http.StatusUnauthorized, "signature mismatch")
			return
		}
		next(w, r)
	}
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type loginReq struct {
	Code string `json:"code"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := h.svc.Login(r.Context(), app.LoginInput{Code: req.Code})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type uploadReq struct {
	UserID string `json:"userId"`
	CatID  string `json:"catId"`
	Suffix string `json:"suffix"`
}

func (h *Handler) createUploadURL(w http.ResponseWriter, r *http.Request) {
	var req uploadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.UserID = userIDFromContext(r.Context())
	out, err := h.svc.CreateUploadSample(r.Context(), app.CreateUploadSampleInput{
		UserID: req.UserID,
		CatID:  req.CatID,
		Suffix: req.Suffix,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out.UploadURL = requestBaseURL(r) + "/v1/samples/upload/" + out.SampleID
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) acceptUpload(w http.ResponseWriter, r *http.Request) {
	sampleID := strings.TrimPrefix(r.URL.Path, "/v1/samples/upload/")
	if sampleID == "" {
		writeError(w, http.StatusBadRequest, "sample id required")
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	if err := os.MkdirAll("/tmp/meowlator/uploads", 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "cannot create upload dir")
		return
	}
	dstPath := filepath.Join("/tmp/meowlator/uploads", sampleID+".jpg")
	dst, err := os.Create(dstPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot persist upload")
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, "cannot write upload")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"sampleId": sampleID, "storedAt": dstPath})
}

type finalizeReq struct {
	SampleID      string                  `json:"sampleId"`
	DeviceCapable bool                    `json:"deviceCapable"`
	SceneTag      string                  `json:"sceneTag"`
	EdgeResult    *domain.InferenceResult `json:"edgeResult"`
	EdgeRuntime   *domain.EdgeRuntime     `json:"edgeRuntime"`
}

func (h *Handler) finalizeInference(w http.ResponseWriter, r *http.Request) {
	var req finalizeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := h.svc.FinalizeInference(r.Context(), app.FinalizeInput{
		SampleID:      req.SampleID,
		DeviceCapable: req.DeviceCapable,
		SceneTag:      req.SceneTag,
		EdgeResult:    req.EdgeResult,
		EdgeRuntime:   req.EdgeRuntime,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type feedbackReq struct {
	SampleID  string `json:"sampleId"`
	UserID    string `json:"userId"`
	IsCorrect bool   `json:"isCorrect"`
	TrueLabel string `json:"trueLabel"`
}

func (h *Handler) saveFeedback(w http.ResponseWriter, r *http.Request) {
	var req feedbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.UserID = userIDFromContext(r.Context())
	label := domain.IntentLabel(req.TrueLabel)
	fb, err := h.svc.SaveFeedback(r.Context(), app.SaveFeedbackInput{
		SampleID:  req.SampleID,
		UserID:    req.UserID,
		IsCorrect: req.IsCorrect,
		TrueLabel: label,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, fb)
}

type generateCopyReq struct {
	Result domain.InferenceResult `json:"result"`
}

func (h *Handler) generateCopy(w http.ResponseWriter, r *http.Request) {
	var req generateCopyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := h.svc.GenerateCopy(r.Context(), req.Result)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) deleteSample(w http.ResponseWriter, r *http.Request) {
	sampleID := strings.TrimPrefix(r.URL.Path, "/v1/samples/")
	if sampleID == "" {
		writeError(w, http.StatusBadRequest, "sample id required")
		return
	}
	if err := h.svc.DeleteSample(r.Context(), sampleID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": sampleID})
}

func (h *Handler) clientConfig(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	cfg := h.svc.ClientConfig(userID)
	writeJSON(w, http.StatusOK, cfg)
}

type rolloutReq struct {
	ModelVersion string  `json:"modelVersion"`
	RolloutRatio float64 `json:"rolloutRatio"`
	TargetBucket int     `json:"targetBucket"`
}

func (h *Handler) rolloutModel(w http.ResponseWriter, r *http.Request) {
	var req rolloutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.svc.RolloutModel(r.Context(), req.ModelVersion, req.RolloutRatio, req.TargetBucket); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"modelVersion": req.ModelVersion, "status": "GRAY", "rolloutRatio": req.RolloutRatio, "targetBucket": req.TargetBucket})
}

type activateReq struct {
	ModelVersion string `json:"modelVersion"`
}

func (h *Handler) activateModel(w http.ResponseWriter, r *http.Request) {
	var req activateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.svc.ActivateModel(r.Context(), req.ModelVersion); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"modelVersion": req.ModelVersion, "status": "ACTIVE"})
}

type registerModelReq struct {
	ModelVersion string          `json:"modelVersion"`
	Metrics      json.RawMessage `json:"metrics"`
}

func (h *Handler) registerModel(w http.ResponseWriter, r *http.Request) {
	var req registerModelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Metrics) == 0 {
		writeError(w, http.StatusBadRequest, "metrics is required")
		return
	}
	if err := h.svc.RegisterModelEvaluation(r.Context(), req.ModelVersion, string(req.Metrics)); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"modelVersion": req.ModelVersion, "status": "CANDIDATE"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrUpstream):
		writeError(w, http.StatusBadGateway, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func userIDFromContext(ctx context.Context) string {
	value := ctx.Value(ctxUserIDKey)
	if value == nil {
		return ""
	}
	userID, _ := value.(string)
	return userID
}

func sessionTokenFromContext(ctx context.Context) string {
	value := ctx.Value(ctxSessionTokenKey)
	if value == nil {
		return ""
	}
	token, _ := value.(string)
	return token
}

func computeRequestSignature(method string, path string, ts string, body string, sessionToken string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(method))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(path))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(ts))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(body))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(sessionToken))
	return fmt.Sprintf("%08x", h.Sum32())
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.ResponseWriter.Write(data)
}

type requestLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
}

type rateBucket struct {
	windowStart time.Time
	count       int
}

func newRequestLimiter() *requestLimiter {
	return &requestLimiter{buckets: map[string]*rateBucket{}}
}

func (l *requestLimiter) Allow(key string, limit int, now time.Time) bool {
	if limit <= 0 {
		return true
	}
	windowStart := now.Truncate(time.Minute)
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket, ok := l.buckets[key]
	if !ok || !bucket.windowStart.Equal(windowStart) {
		l.buckets[key] = &rateBucket{windowStart: windowStart, count: 1}
		return true
	}
	if bucket.count >= limit {
		return false
	}
	bucket.count++
	return true
}
