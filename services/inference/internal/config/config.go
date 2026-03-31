package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr              string
	PredictorMode     string
	ModelPriorsPath   string
	ONNXModelPath     string
	ONNXSharedLibPath string
	UploadRoot        string
	ONNXInputSize     int
}

func Load() Config {
	addr := os.Getenv("INFERENCE_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	uploadRoot := os.Getenv("INFERENCE_UPLOAD_ROOT")
	if uploadRoot == "" {
		uploadRoot = os.Getenv("UPLOAD_ROOT")
	}
	if uploadRoot == "" {
		uploadRoot = "/tmp/meowlator/uploads"
	}
	inputSize := 224
	if raw := strings.TrimSpace(os.Getenv("ONNX_INPUT_SIZE")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			inputSize = value
		}
	}
	return Config{
		Addr:              addr,
		PredictorMode:     strings.ToLower(strings.TrimSpace(os.Getenv("INFERENCE_PREDICTOR_MODE"))),
		ModelPriorsPath:   os.Getenv("MODEL_PRIORS_PATH"),
		ONNXModelPath:     strings.TrimSpace(os.Getenv("ONNX_MODEL_PATH")),
		ONNXSharedLibPath: strings.TrimSpace(os.Getenv("ONNX_SHARED_LIB_PATH")),
		UploadRoot:        uploadRoot,
		ONNXInputSize:     inputSize,
	}
}
