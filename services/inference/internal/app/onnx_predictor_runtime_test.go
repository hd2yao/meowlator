package app

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
	for _, candidate := range []string{
		filepath.Join(moduleDir, "test_data", "onnxruntime_arm64.dylib"),
		filepath.Join(moduleDir, "test_data", "onnxruntime_arm64.so"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatalf("failed to find onnxruntime shared library test fixture under %s/test_data", moduleDir)
	return ""
}

func writeConstantONNXModel(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "const.onnx")
	script := `
import sys
import onnx
from onnx import helper, TensorProto

path = sys.argv[1]
x = helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 3, 4, 4])
y = helper.make_tensor_value_info("logits", TensorProto.FLOAT, [1, 8])
const = helper.make_tensor("value", TensorProto.FLOAT, [1, 8], [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8])
node = helper.make_node("Constant", inputs=[], outputs=["logits"], value=const)
graph = helper.make_graph([node], "const-model", [x], [y])
model = helper.make_model(graph, opset_imports=[helper.make_opsetid("", 17)])
onnx.save(model, path)
`
	runCmd(t, "python3", "-c", script, path)
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
