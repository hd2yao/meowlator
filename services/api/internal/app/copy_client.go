package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type CopyClientConfig struct {
	Timeout time.Duration
}

type CopyHTTPClient struct {
	endpoint string
	enabled  bool
	client   *http.Client
}

func NewCopyClient(cfg CopyClientConfig) *CopyHTTPClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 1200 * time.Millisecond
	}
	enabled := strings.EqualFold(os.Getenv("COPY_LLM_ENABLED"), "true")
	endpoint := os.Getenv("COPY_LLM_ENDPOINT")
	return &CopyHTTPClient{
		endpoint: endpoint,
		enabled:  enabled && endpoint != "",
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

type copyRequest struct {
	Result       domain.InferenceResult `json:"result"`
	StyleVersion string                 `json:"styleVersion"`
}

type llmResponse struct {
	CatLine    string `json:"catLine"`
	Evidence   string `json:"evidence"`
	ShareTitle string `json:"shareTitle"`
}

func (c *CopyHTTPClient) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	if c.enabled {
		block, err := c.generateFromLLM(ctx, result, styleVersion)
		if err == nil {
			return block, nil
		}
	}
	return generateTemplateCopy(result), nil
}

func (c *CopyHTTPClient) generateFromLLM(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	payload := copyRequest{Result: result, StyleVersion: styleVersion}
	raw, err := json.Marshal(payload)
	if err != nil {
		return domain.CopyBlock{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.CopyBlock{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return domain.CopyBlock{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return domain.CopyBlock{}, fmt.Errorf("llm copy status: %d", resp.StatusCode)
	}
	var parsed llmResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return domain.CopyBlock{}, err
	}
	if parsed.CatLine == "" || parsed.Evidence == "" || parsed.ShareTitle == "" {
		return domain.CopyBlock{}, errors.New("missing llm fields")
	}
	return domain.CopyBlock{CatLine: parsed.CatLine, Evidence: parsed.Evidence, ShareTitle: parsed.ShareTitle}, nil
}

func ParseCopyJSON(raw string) (domain.CopyBlock, error) {
	var parsed llmResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return domain.CopyBlock{}, err
	}
	if parsed.CatLine == "" || parsed.Evidence == "" || parsed.ShareTitle == "" {
		return domain.CopyBlock{}, errors.New("invalid copy response")
	}
	return domain.CopyBlock{CatLine: parsed.CatLine, Evidence: parsed.Evidence, ShareTitle: parsed.ShareTitle}, nil
}

func generateTemplateCopy(result domain.InferenceResult) domain.CopyBlock {
	top := "UNCERTAIN"
	prob := 0.0
	if len(result.IntentTop3) > 0 {
		top = string(result.IntentTop3[0].Label)
		prob = result.IntentTop3[0].Prob
	}
	catLine := fmt.Sprintf("喵喵委员会投票 %.0f%%：本猫当前诉求是 %s，速速响应。", prob*100, top)
	evidence := fmt.Sprintf("依据：状态(紧张%s/兴奋%s/舒适%s) + 视觉证据 %s。", result.State.Tension, result.State.Arousal, result.State.Comfort, strings.Join(result.Evidence, "、"))
	title := fmt.Sprintf("主子发话：%s（翻译版）", top)
	return domain.CopyBlock{CatLine: catLine, Evidence: evidence, ShareTitle: title}
}
