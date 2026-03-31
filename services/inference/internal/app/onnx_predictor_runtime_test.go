package app

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const constantLogitsONNXBase64 = "CA06mQEKTxIGbG9naXRzIghDb25zdGFudCo7CgV2YWx1ZSovCAEICBABIiDNzMw9zcxMPpqZmT7NzMw+AAAAP5qZGT8zMzM/zcxMP0IFdmFsdWWgAQQSC2NvbnN0LW1vZGVsWh8KBWlucHV0EhYKFAgBEhAKAggBCgIIAwoCCAQKAggEYhgKBmxvZ2l0cxIOCgwIARIICgIIAQoCCAhCBAoAEBE="

func TestONNXPredictorRunsRealSession(t *testing.T) {
	predictor, err := NewPredictor(PredictorConfig{
		Mode:          "onnx",
		ModelPath:     writeConstantONNXModel(t),
		SharedLibPath: testSharedLibPath(t),
		UploadRoot:    writeUploadFixture(t),
		InputSize:     4,
	})
	if err != nil {
		t.Fatalf("new predictor failed: %v", err)
	}

	result, err := predictor.Predict("samples/u1/sample-123.jpg", "UNKNOWN")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if result.Source != "CLOUD" {
		t.Fatalf("expected CLOUD source, got %s", result.Source)
	}
	if len(result.IntentTop3) != 3 {
		t.Fatalf("expected top3 intents, got %d", len(result.IntentTop3))
	}
	if result.IntentTop3[0].Label != IntentUncertain {
		t.Fatalf("expected top intent %s, got %s", IntentUncertain, result.IntentTop3[0].Label)
	}
}

func testSharedLibPath(t *testing.T) string {
	t.Helper()
	if path := strings.TrimSpace(os.Getenv("ONNX_SHARED_LIB_PATH")); path != "" {
		return path
	}
	moduleDir := strings.TrimSpace(runCmd(t, "go", "list", "-f", "{{.Dir}}", "-m", "github.com/yalue/onnxruntime_go"))
	var candidates []string
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		candidates = []string{filepath.Join(moduleDir, "test_data", "onnxruntime_arm64.dylib")}
	case "linux/arm64":
		candidates = []string{filepath.Join(moduleDir, "test_data", "onnxruntime_arm64.so")}
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Skipf("no onnxruntime shared library fixture for %s/%s", runtime.GOOS, runtime.GOARCH)
	return ""
}

func writeConstantONNXModel(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "const.onnx")
	raw, err := base64.StdEncoding.DecodeString(constantLogitsONNXBase64)
	if err != nil {
		t.Fatalf("decode embedded onnx fixture failed: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write embedded onnx fixture failed: %v", err)
	}
	return path
}

func writeUploadFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: uint8(10 * x), G: uint8(20 * y), B: 100, A: 255})
		}
	}
	file, err := os.Create(filepath.Join(root, "sample-123.jpg"))
	if err != nil {
		t.Fatalf("create fixture jpg failed: %v", err)
	}
	defer file.Close()
	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode fixture jpg failed: %v", err)
	}
	return root
}

func runCmd(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, output)
	}
	return string(output)
}
