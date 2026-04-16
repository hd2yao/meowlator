package app

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
	"golang.org/x/image/draw"
)

const defaultUploadRoot = "/tmp/meowlator/uploads"

var ortInitMu sync.Mutex

type ONNXPredictor struct {
	uploadRoot string
	inputSize  int
	priors     map[IntentLabel]float64
	session    *ort.DynamicAdvancedSession
}

func NewONNXPredictor(cfg PredictorConfig) (*ONNXPredictor, error) {
	uploadRoot := strings.TrimSpace(cfg.UploadRoot)
	if uploadRoot == "" {
		uploadRoot = defaultUploadRoot
	}
	inputSize := cfg.InputSize
	if inputSize <= 0 {
		inputSize = 224
	}

	if err := initializeORT(cfg.SharedLibPath); err != nil {
		return nil, err
	}
	session, err := ort.NewDynamicAdvancedSession(cfg.ModelPath, []string{"input"}, []string{"logits"}, nil)
	if err != nil {
		return nil, fmt.Errorf("create onnx session: %w", err)
	}

	return &ONNXPredictor{
		uploadRoot: uploadRoot,
		inputSize:  inputSize,
		priors:     normalizePriors(cfg.Priors),
		session:    session,
	}, nil
}

func initializeORT(sharedLibPath string) error {
	ortInitMu.Lock()
	defer ortInitMu.Unlock()

	if ort.IsInitialized() {
		return nil
	}
	ort.SetSharedLibraryPath(sharedLibPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("initialize onnx runtime: %w", err)
	}
	return nil
}

func (p *ONNXPredictor) Name() string {
	return "onnx"
}

func (p *ONNXPredictor) Predict(imageKey string, sceneTag string) (InferenceResult, error) {
	imagePath, err := resolveImagePath(p.uploadRoot, imageKey)
	if err != nil {
		return InferenceResult{}, err
	}
	inputData, err := loadImageTensor(imagePath, p.inputSize)
	if err != nil {
		return InferenceResult{}, err
	}

	input, err := ort.NewTensor(ort.NewShape(1, 3, int64(p.inputSize), int64(p.inputSize)), inputData)
	if err != nil {
		return InferenceResult{}, fmt.Errorf("create input tensor: %w", err)
	}
	defer input.Destroy()

	outputs := []ort.Value{nil}
	if err := p.session.Run([]ort.Value{input}, outputs); err != nil {
		return InferenceResult{}, fmt.Errorf("run onnx session: %w", err)
	}
	if len(outputs) != 1 || outputs[0] == nil {
		return InferenceResult{}, fmt.Errorf("onnx session returned no outputs")
	}

	output, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return InferenceResult{}, fmt.Errorf("unexpected onnx output type %T", outputs[0])
	}
	defer output.Destroy()

	return p.buildResult(imageKey, sceneTag, output.GetData())
}

func resolveImagePath(uploadRoot string, imageKey string) (string, error) {
	trimmed := strings.TrimSpace(imageKey)
	if trimmed == "" {
		return "", fmt.Errorf("imageKey is required")
	}
	base := filepath.Base(trimmed)
	if base == "." || base == "" || !strings.Contains(base, ".") {
		return "", fmt.Errorf("invalid imageKey: %s", imageKey)
	}
	ext := filepath.Ext(base)
	sampleID := strings.TrimSuffix(base, ext)
	if sampleID == "" {
		return "", fmt.Errorf("invalid imageKey: %s", imageKey)
	}
	// API uploads are persisted as <sampleId>.jpg regardless of the original key suffix.
	fullPath := filepath.Join(uploadRoot, sampleID+".jpg")
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("image file not found for imageKey %s", imageKey)
		}
		return "", fmt.Errorf("stat image path: %w", err)
	}
	return fullPath, nil
}

func loadImageTensor(imagePath string, size int) ([]float32, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()

	src, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	pixels := size * size
	data := make([]float32, 3*pixels)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r, g, b, _ := dst.At(x, y).RGBA()
			idx := y*size + x
			data[idx] = normalizeChannel(r, 0.485, 0.229)
			data[pixels+idx] = normalizeChannel(g, 0.456, 0.224)
			data[(2*pixels)+idx] = normalizeChannel(b, 0.406, 0.225)
		}
	}
	return data, nil
}

func normalizeChannel(v uint32, mean float32, std float32) float32 {
	value := float32(v>>8) / 255.0
	return (value - mean) / std
}

func (p *ONNXPredictor) buildResult(imageKey string, sceneTag string, logits []float32) (InferenceResult, error) {
	if len(logits) < len(allIntents) {
		return InferenceResult{}, fmt.Errorf("onnx logits too short: got %d want at least %d", len(logits), len(allIntents))
	}

	probs := softmax(logits[:len(allIntents)])
	candidates := make([]IntentProb, 0, len(allIntents))
	for idx, label := range allIntents {
		prob := probs[idx]
		if prior, ok := p.priors[label]; ok {
			prob = clamp((1.0-priorBlendWeight)*prob+priorBlendWeight*prior, 0.0, 1.0)
		}
		candidates = append(candidates, IntentProb{Label: label, Prob: prob})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Prob > candidates[j].Prob
	})

	top3 := append([]IntentProb(nil), candidates[:3]...)
	return InferenceResult{
		IntentTop3:       top3,
		State:            deriveState(hashValue(imageKey + "|" + sceneTag)),
		Confidence:       top3[0].Prob,
		Source:           "CLOUD",
		Evidence:         []string{"云端 ONNX 复判", "视觉 logits 排序"},
		CopyStyleVersion: "v1",
	}, nil
}

func softmax(logits []float32) []float64 {
	maxLogit := logits[0]
	for _, value := range logits[1:] {
		if value > maxLogit {
			maxLogit = value
		}
	}
	result := make([]float64, len(logits))
	total := 0.0
	for idx, value := range logits {
		expValue := math.Exp(float64(value - maxLogit))
		result[idx] = expValue
		total += expValue
	}
	for idx := range result {
		result[idx] /= total
	}
	return result
}

func toPercent3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
