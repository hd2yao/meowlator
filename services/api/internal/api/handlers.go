package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dysania/meowlator/services/api/internal/app"
	"github.com/dysania/meowlator/services/api/internal/domain"
)

type Handler struct {
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /v1/samples/upload-url", h.createUploadURL)
	mux.HandleFunc("POST /v1/samples/upload/", h.acceptUpload)
	mux.HandleFunc("POST /v1/inference/finalize", h.finalizeInference)
	mux.HandleFunc("POST /v1/feedback", h.saveFeedback)
	mux.HandleFunc("POST /v1/copy/generate", h.generateCopy)
	mux.HandleFunc("DELETE /v1/samples/", h.deleteSample)
	mux.HandleFunc("GET /v1/metrics/client-config", h.clientConfig)
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	userID := r.URL.Query().Get("userId")
	cfg := h.svc.ClientConfig(userID)
	writeJSON(w, http.StatusOK, cfg)
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
