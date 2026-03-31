package app

import (
	"strings"
	"testing"
)

func TestNewPredictorRejectsOnnxWithoutModelPath(t *testing.T) {
	_, err := NewPredictor(PredictorConfig{
		Mode:          "onnx",
		SharedLibPath: "/tmp/libonnxruntime.so.1.24.1",
		UploadRoot:    "/tmp/meowlator/uploads",
	})
	if err == nil {
		t.Fatalf("expected missing model path error")
	}
	if !strings.Contains(err.Error(), "ONNX_MODEL_PATH") {
		t.Fatalf("expected ONNX_MODEL_PATH error, got %v", err)
	}
}

func TestNewPredictorRejectsOnnxWithoutSharedLib(t *testing.T) {
	_, err := NewPredictor(PredictorConfig{
		Mode:       "onnx",
		ModelPath:  "/tmp/model.onnx",
		UploadRoot: "/tmp/meowlator/uploads",
	})
	if err == nil {
		t.Fatalf("expected missing shared lib error")
	}
	if !strings.Contains(err.Error(), "ONNX_SHARED_LIB_PATH") {
		t.Fatalf("expected ONNX_SHARED_LIB_PATH error, got %v", err)
	}
}

func TestResolveImagePathRejectsInvalidImageKey(t *testing.T) {
	_, err := resolveImagePath("/tmp/meowlator/uploads", "bad-key")
	if err == nil {
		t.Fatalf("expected invalid imageKey error")
	}
}

func TestNormalizeChannelUsesImageNetStats(t *testing.T) {
	got := normalizeChannel(124 << 8)
	want := float32((124.0/255.0 - 0.485) / 0.229)
	if diff := got - want; diff < -0.0001 || diff > 0.0001 {
		t.Fatalf("expected imagenet normalization %.6f, got %.6f", want, got)
	}
}
