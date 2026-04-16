package app

import (
	"fmt"
	"strings"
)

type Predictor interface {
	Predict(imageKey string, sceneTag string) (InferenceResult, error)
	Name() string
}

type PredictorConfig struct {
	Mode          string
	ModelPath     string
	SharedLibPath string
	UploadRoot    string
	Priors        map[IntentLabel]float64
	InputSize     int
}

func NewPredictor(cfg PredictorConfig) (Predictor, error) {
	mode := strings.TrimSpace(cfg.Mode)
	if mode == "" {
		mode = "heuristic"
	}
	switch mode {
	case "heuristic":
		return NewModel(cfg.Priors), nil
	case "onnx":
		if strings.TrimSpace(cfg.ModelPath) == "" {
			return nil, fmt.Errorf("ONNX_MODEL_PATH is required when INFERENCE_PREDICTOR_MODE=onnx")
		}
		if strings.TrimSpace(cfg.SharedLibPath) == "" {
			return nil, fmt.Errorf("ONNX_SHARED_LIB_PATH is required when INFERENCE_PREDICTOR_MODE=onnx")
		}
		return NewONNXPredictor(cfg)
	default:
		return nil, fmt.Errorf("unsupported INFERENCE_PREDICTOR_MODE: %s", mode)
	}
}

func (m *Model) Name() string {
	return "heuristic"
}
