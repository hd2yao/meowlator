package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dysania/meowlator/services/inference/internal/api"
	"github.com/dysania/meowlator/services/inference/internal/app"
	"github.com/dysania/meowlator/services/inference/internal/config"
)

func main() {
	cfg := config.Load()
	if err := prepareRuntime(cfg); err != nil {
		log.Fatal(err)
	}

	priors, err := app.LoadIntentPriors(cfg.ModelPriorsPath)
	if err != nil {
		log.Printf("failed to load priors from %s, fallback to default predictor: %v", cfg.ModelPriorsPath, err)
	}
	if len(priors) > 0 {
		log.Printf("loaded intent priors from %s", cfg.ModelPriorsPath)
	}

	predictor, err := app.NewPredictor(app.PredictorConfig{
		Mode:          cfg.PredictorMode,
		ModelPath:     cfg.ONNXModelPath,
		SharedLibPath: cfg.ONNXSharedLibPath,
		UploadRoot:    cfg.UploadRoot,
		Priors:        priors,
		InputSize:     cfg.ONNXInputSize,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("inference predictor mode=%s", predictor.Name())
	h := api.NewHandler(predictor)
	mux := http.NewServeMux()
	h.Register(mux)

	log.Printf("inference service listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}

func prepareRuntime(cfg config.Config) error {
	if cfg.UploadRoot == "" {
		return fmt.Errorf("upload root is required")
	}
	if err := os.MkdirAll(cfg.UploadRoot, 0o755); err != nil {
		return fmt.Errorf("prepare upload root %s: %w", cfg.UploadRoot, err)
	}
	if cfg.PredictorMode != "onnx" {
		return nil
	}
	if cfg.ONNXInputSize <= 0 {
		return fmt.Errorf("onnx input size must be positive")
	}
	if cfg.ONNXModelPath == "" {
		return fmt.Errorf("onnx model path is required when predictor mode is onnx")
	}
	if _, err := os.Stat(cfg.ONNXModelPath); err != nil {
		return fmt.Errorf("stat onnx model path %s: %w", cfg.ONNXModelPath, err)
	}
	if cfg.ONNXSharedLibPath == "" {
		return fmt.Errorf("onnx shared library path is required when predictor mode is onnx")
	}
	if _, err := os.Stat(cfg.ONNXSharedLibPath); err != nil {
		return fmt.Errorf("stat onnx shared library path %s: %w", cfg.ONNXSharedLibPath, err)
	}
	return nil
}
