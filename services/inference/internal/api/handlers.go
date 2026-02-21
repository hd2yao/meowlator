package api

import (
	"encoding/json"
	"net/http"

	"github.com/dysania/meowlator/services/inference/internal/app"
)

type Handler struct {
	model *app.Model
}

func NewHandler(model *app.Model) *Handler {
	return &Handler{model: model}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /v1/inference/predict", h.predict)
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type predictReq struct {
	ImageKey string `json:"imageKey"`
	SceneTag string `json:"sceneTag"`
}

type predictResp struct {
	Result app.InferenceResult `json:"result"`
}

func (h *Handler) predict(w http.ResponseWriter, r *http.Request) {
	var req predictReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ImageKey == "" {
		writeError(w, http.StatusBadRequest, "imageKey is required")
		return
	}
	result := h.model.Predict(req.ImageKey, req.SceneTag)
	writeJSON(w, http.StatusOK, predictResp{Result: result})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
