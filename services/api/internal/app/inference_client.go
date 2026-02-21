package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type HTTPInferenceClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPInferenceClient(baseURL string) *HTTPInferenceClient {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &HTTPInferenceClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 1500 * time.Millisecond,
		},
	}
}

type predictRequest struct {
	ImageKey string `json:"imageKey"`
	SceneTag string `json:"sceneTag"`
}

type predictResponse struct {
	Result domain.InferenceResult `json:"result"`
}

func (c *HTTPInferenceClient) Predict(ctx context.Context, imageKey string, sceneTag string) (*domain.InferenceResult, error) {
	payload := predictRequest{ImageKey: imageKey, SceneTag: sceneTag}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/inference/predict", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("inference service status: %d", resp.StatusCode)
	}

	var out predictResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out.Result, nil
}
