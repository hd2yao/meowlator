package app

import (
	"context"
	"testing"

	"github.com/dysania/meowlator/services/api/internal/domain"
	"github.com/dysania/meowlator/services/api/internal/repository"
)

type fakeInference struct {
	result domain.InferenceResult
	err    error
}

func (f fakeInference) Predict(ctx context.Context, imageKey string, sceneTag string) (*domain.InferenceResult, error) {
	_ = ctx
	_ = imageKey
	_ = sceneTag
	if f.err != nil {
		return nil, f.err
	}
	res := f.result
	return &res, nil
}

type fakeCopy struct{}

func (f fakeCopy) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	_ = ctx
	_ = result
	_ = styleVersion
	return domain.CopyBlock{CatLine: "喵", Evidence: "依据", ShareTitle: "标题"}, nil
}

func TestFinalizeInferenceCloudFallback(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewService(repo, fakeInference{result: domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentWantPlay, Prob: 0.8}},
		State:      domain.State3D{Tension: domain.LevelMid, Arousal: domain.LevelHigh, Comfort: domain.LevelLow},
		Confidence: 0.8,
		Source:     "CLOUD",
	}}, fakeCopy{}, Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45}, 7, "model-v1", false)

	upload, err := svc.CreateUploadSample(context.Background(), CreateUploadSampleInput{UserID: "u1", CatID: "c1"})
	if err != nil {
		t.Fatalf("create sample failed: %v", err)
	}

	edge := &domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentFeeding, Prob: 0.55}},
		State:      domain.State3D{Tension: domain.LevelMid, Arousal: domain.LevelMid, Comfort: domain.LevelLow},
		Confidence: 0.55,
		Source:     "EDGE",
	}
	out, err := svc.FinalizeInference(context.Background(), FinalizeInput{SampleID: upload.SampleID, DeviceCapable: true, EdgeResult: edge})
	if err != nil {
		t.Fatalf("finalize failed: %v", err)
	}
	if out.Result.Source != "CLOUD" {
		t.Fatalf("expected cloud fallback, got %s", out.Result.Source)
	}
}

func TestFinalizeInferenceEdgeRuntimeMeta(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewService(repo, fakeInference{result: domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentWantPlay, Prob: 0.8}},
		State:      domain.State3D{Tension: domain.LevelMid, Arousal: domain.LevelHigh, Comfort: domain.LevelLow},
		Confidence: 0.8,
		Source:     "CLOUD",
	}}, fakeCopy{}, Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45}, 7, "model-v1", false)

	upload, err := svc.CreateUploadSample(context.Background(), CreateUploadSampleInput{UserID: "u1", CatID: "c1"})
	if err != nil {
		t.Fatalf("create sample failed: %v", err)
	}

	edge := &domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentFeeding, Prob: 0.82}},
		State:      domain.State3D{Tension: domain.LevelMid, Arousal: domain.LevelMid, Comfort: domain.LevelLow},
		Confidence: 0.82,
		Source:     "EDGE",
	}
	out, err := svc.FinalizeInference(context.Background(), FinalizeInput{
		SampleID:      upload.SampleID,
		DeviceCapable: true,
		EdgeResult:    edge,
		EdgeRuntime: &domain.EdgeRuntime{
			Engine:       "edge-onnx-v1",
			ModelVersion: "mobilenetv3-small-int8-v2",
			LoadMS:       55,
			InferMS:      38,
			DeviceModel:  "iPhone15,2",
		},
	})
	if err != nil {
		t.Fatalf("finalize failed: %v", err)
	}
	if out.Result.Source != "EDGE" {
		t.Fatalf("expected edge result, got %s", out.Result.Source)
	}
	if out.Result.EdgeMeta == nil {
		t.Fatalf("expected edgeMeta to be set")
	}
	if out.Result.EdgeMeta.FallbackUsed {
		t.Fatalf("expected no fallback for high confidence edge result")
	}
	if !out.Result.EdgeMeta.UsedEdgeResult {
		t.Fatalf("expected UsedEdgeResult to be true")
	}
}

func TestFinalizeInferenceRejectInvalidEdgeRuntime(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewService(repo, fakeInference{}, fakeCopy{}, Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45}, 7, "model-v1", false)
	upload, err := svc.CreateUploadSample(context.Background(), CreateUploadSampleInput{UserID: "u1", CatID: "c1"})
	if err != nil {
		t.Fatalf("create sample failed: %v", err)
	}
	_, err = svc.FinalizeInference(context.Background(), FinalizeInput{
		SampleID:      upload.SampleID,
		DeviceCapable: true,
		EdgeRuntime: &domain.EdgeRuntime{
			Engine:       "",
			ModelVersion: "m",
			LoadMS:       -1,
			InferMS:      10,
			DeviceModel:  "dev",
		},
	})
	if err == nil {
		t.Fatalf("expected invalid edgeRuntime error")
	}
}

func TestFinalizeInferenceWithPainRisk(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewService(repo, fakeInference{}, fakeCopy{}, Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45}, 7, "model-v1", true)
	upload, err := svc.CreateUploadSample(context.Background(), CreateUploadSampleInput{UserID: "u1", CatID: "c1"})
	if err != nil {
		t.Fatalf("create sample failed: %v", err)
	}

	edge := &domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentDefensiveAlert, Prob: 0.82}},
		State:      domain.State3D{Tension: domain.LevelHigh, Arousal: domain.LevelHigh, Comfort: domain.LevelLow},
		Confidence: 0.82,
		Source:     "EDGE",
	}
	out, err := svc.FinalizeInference(context.Background(), FinalizeInput{
		SampleID:      upload.SampleID,
		DeviceCapable: true,
		EdgeResult:    edge,
	})
	if err != nil {
		t.Fatalf("finalize failed: %v", err)
	}
	if out.Result.Risk == nil {
		t.Fatalf("expected risk info to be populated")
	}
	if out.Result.Risk.PainRiskLevel != domain.PainRiskHigh {
		t.Fatalf("expected high pain risk, got %s", out.Result.Risk.PainRiskLevel)
	}
	if out.Result.Risk.Disclaimer == "" {
		t.Fatalf("expected risk disclaimer")
	}
}

func TestSaveFeedback(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewService(repo, fakeInference{}, fakeCopy{}, Thresholds{EdgeAccept: 0.7, CloudFallback: 0.45}, 7, "model-v1", false)

	upload, err := svc.CreateUploadSample(context.Background(), CreateUploadSampleInput{UserID: "u1", CatID: "c1"})
	if err != nil {
		t.Fatalf("create sample failed: %v", err)
	}
	fb, err := svc.SaveFeedback(context.Background(), SaveFeedbackInput{
		SampleID:  upload.SampleID,
		UserID:    "u1",
		IsCorrect: false,
		TrueLabel: domain.IntentSeekAttention,
	})
	if err != nil {
		t.Fatalf("save feedback failed: %v", err)
	}
	if fb.TrainingWeight != domain.WeightCorrected {
		t.Fatalf("expected corrected weight %.1f got %.1f", domain.WeightCorrected, fb.TrainingWeight)
	}
}
