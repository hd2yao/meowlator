package app

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/dysania/meowlator/services/api/internal/domain"
	"github.com/dysania/meowlator/services/api/internal/repository"
)

type Repository interface {
	CreateSample(ctx context.Context, sample *domain.Sample) error
	GetSample(ctx context.Context, sampleID string) (*domain.Sample, error)
	UpdatePredictions(ctx context.Context, sampleID string, edgePred, finalPred *domain.InferenceResult, sceneTag string, modelVersion string) error
	SaveFeedback(ctx context.Context, fb *domain.Feedback) error
	DeleteSample(ctx context.Context, sampleID string) error
	UserFeedbackStats(ctx context.Context, userID string) (total int, extremeRatio float64, suspicious bool)
}

type InferenceClient interface {
	Predict(ctx context.Context, imageKey string, sceneTag string) (*domain.InferenceResult, error)
}

type CopyClient interface {
	Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error)
}

type Thresholds struct {
	EdgeAccept    float64
	CloudFallback float64
}

type Service struct {
	repo          Repository
	inference     InferenceClient
	copyClient    CopyClient
	thresholds    Thresholds
	retentionDays int
	modelVersion  string
	painRisk      bool
}

func NewService(repo Repository, inference InferenceClient, copyClient CopyClient, thresholds Thresholds, retentionDays int, modelVersion string, painRisk bool) *Service {
	if retentionDays <= 0 {
		retentionDays = 7
	}
	if modelVersion == "" {
		modelVersion = "mobilenetv3-small-int8-v1"
	}
	return &Service{
		repo:          repo,
		inference:     inference,
		copyClient:    copyClient,
		thresholds:    thresholds,
		retentionDays: retentionDays,
		modelVersion:  modelVersion,
		painRisk:      painRisk,
	}
}

type CreateUploadSampleInput struct {
	UserID string
	CatID  string
	Suffix string
}

type CreateUploadSampleOutput struct {
	SampleID          string `json:"sampleId"`
	ImageKey          string `json:"imageKey"`
	UploadURL         string `json:"uploadUrl"`
	ExpiresInSeconds  int    `json:"expiresInSeconds"`
	RetentionDeadline int64  `json:"retentionDeadline"`
}

func (s *Service) CreateUploadSample(ctx context.Context, in CreateUploadSampleInput) (*CreateUploadSampleOutput, error) {
	if in.UserID == "" {
		return nil, fmt.Errorf("%w: userId is required", domain.ErrBadRequest)
	}
	if in.CatID == "" {
		in.CatID = "cat-default"
	}
	sampleID := repository.GenerateID("sample")
	imageKey := fmt.Sprintf("samples/%s/%s%s", in.UserID, sampleID, normalizeSuffix(in.Suffix))
	now := time.Now()
	expireAt := now.Add(time.Duration(s.retentionDays) * 24 * time.Hour)
	sample := &domain.Sample{
		SampleID:     sampleID,
		UserID:       in.UserID,
		CatID:        in.CatID,
		ImageKey:     imageKey,
		ModelVersion: s.modelVersion,
		CreatedAt:    now.Unix(),
		ExpireAt:     expireAt.Unix(),
	}
	if err := s.repo.CreateSample(ctx, sample); err != nil {
		return nil, err
	}
	mockUploadURL := (&url.URL{Scheme: "https", Host: "upload.example.local", Path: "/put"}).String() + "?key=" + url.QueryEscape(imageKey)
	return &CreateUploadSampleOutput{
		SampleID:          sampleID,
		ImageKey:          imageKey,
		UploadURL:         mockUploadURL,
		ExpiresInSeconds:  600,
		RetentionDeadline: expireAt.Unix(),
	}, nil
}

type FinalizeInput struct {
	SampleID      string
	DeviceCapable bool
	SceneTag      string
	EdgeResult    *domain.InferenceResult
	EdgeRuntime   *domain.EdgeRuntime
}

type FinalizeOutput struct {
	SampleID     string                 `json:"sampleId"`
	Result       domain.InferenceResult `json:"result"`
	Copy         domain.CopyBlock       `json:"copy"`
	NeedFeedback bool                   `json:"needFeedback"`
	FallbackUsed bool                   `json:"fallbackUsed"`
}

func (s *Service) FinalizeInference(ctx context.Context, in FinalizeInput) (*FinalizeOutput, error) {
	sample, err := s.repo.GetSample(ctx, in.SampleID)
	if err != nil {
		return nil, err
	}
	if err := in.EdgeRuntime.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrBadRequest, err)
	}

	var finalResult domain.InferenceResult
	fallbackUsed := false
	forceFeedback := false

	if in.EdgeResult != nil {
		in.EdgeResult.Source = "EDGE"
		in.EdgeResult.NormalizeTopK(3)
		if err := in.EdgeResult.Validate(); err != nil {
			return nil, fmt.Errorf("%w: invalid edge result: %v", domain.ErrBadRequest, err)
		}
		decision := domain.DecideThreshold(in.EdgeResult.Confidence, in.DeviceCapable, domain.ThresholdConfig{
			EdgeAccept:    s.thresholds.EdgeAccept,
			CloudFallback: s.thresholds.CloudFallback,
		})
		forceFeedback = decision.ForceFeedback
		if decision.UseEdge {
			finalResult = *in.EdgeResult
		} else {
			fallbackUsed = true
		}
	} else {
		fallbackUsed = true
		forceFeedback = true
	}

	if fallbackUsed {
		cloudResult, cloudErr := s.inference.Predict(ctx, sample.ImageKey, in.SceneTag)
		if cloudErr == nil {
			cloudResult.Source = "CLOUD"
			cloudResult.NormalizeTopK(3)
			if err := cloudResult.Validate(); err == nil {
				finalResult = *cloudResult
			}
		}
		if len(finalResult.IntentTop3) == 0 && in.EdgeResult != nil {
			finalResult = *in.EdgeResult
			finalResult.Source = "EDGE"
			forceFeedback = true
		}
		if len(finalResult.IntentTop3) == 0 {
			return nil, fmt.Errorf("%w: no available inference result", domain.ErrUpstream)
		}
	}

	if finalResult.Confidence < s.thresholds.CloudFallback {
		forceFeedback = true
	}
	if in.EdgeRuntime != nil {
		finalResult.EdgeMeta = &domain.EdgeMeta{
			Engine:         in.EdgeRuntime.Engine,
			ModelVersion:   in.EdgeRuntime.ModelVersion,
			LoadMS:         in.EdgeRuntime.LoadMS,
			InferMS:        in.EdgeRuntime.InferMS,
			DeviceModel:    in.EdgeRuntime.DeviceModel,
			FailureReason:  in.EdgeRuntime.FailureReason,
			FallbackUsed:   fallbackUsed,
			UsedEdgeResult: finalResult.Source == "EDGE",
		}
	}
	if s.painRisk {
		finalResult.Risk = domain.EvaluatePainRisk(finalResult)
	}

	finalResult.CopyStyleVersion = "v1"
	copyBlock, err := s.copyClient.Generate(ctx, finalResult, finalResult.CopyStyleVersion)
	if err != nil {
		return nil, fmt.Errorf("%w: copy generation failed", domain.ErrUpstream)
	}

	if err := s.repo.UpdatePredictions(ctx, in.SampleID, in.EdgeResult, &finalResult, in.SceneTag, s.modelVersion); err != nil {
		return nil, err
	}

	return &FinalizeOutput{
		SampleID:     in.SampleID,
		Result:       finalResult,
		Copy:         copyBlock,
		NeedFeedback: forceFeedback,
		FallbackUsed: fallbackUsed,
	}, nil
}

type SaveFeedbackInput struct {
	SampleID  string
	UserID    string
	IsCorrect bool
	TrueLabel domain.IntentLabel
}

func (s *Service) SaveFeedback(ctx context.Context, in SaveFeedbackInput) (*domain.Feedback, error) {
	if in.SampleID == "" || in.UserID == "" {
		return nil, fmt.Errorf("%w: sampleId and userId are required", domain.ErrBadRequest)
	}
	if !in.IsCorrect {
		if _, err := domain.ParseIntentLabel(string(in.TrueLabel)); err != nil {
			return nil, fmt.Errorf("%w: trueLabel is required when isCorrect=false", domain.ErrBadRequest)
		}
	}

	total, extremeRatio, suspicious := s.repo.UserFeedbackStats(ctx, in.UserID)
	score := domain.ReliabilityScore(total, extremeRatio, suspicious)
	weight := domain.FeedbackWeight(in.IsCorrect)

	fb := &domain.Feedback{
		FeedbackID:       repository.GenerateID("fb"),
		SampleID:         in.SampleID,
		UserID:           in.UserID,
		IsCorrect:        in.IsCorrect,
		TrueLabel:        in.TrueLabel,
		ReliabilityScore: score,
		TrainingWeight:   weight,
		CreatedAt:        time.Now().Unix(),
	}
	if err := s.repo.SaveFeedback(ctx, fb); err != nil {
		return nil, err
	}
	return fb, nil
}

func (s *Service) DeleteSample(ctx context.Context, sampleID string) error {
	if sampleID == "" {
		return fmt.Errorf("%w: sampleId is required", domain.ErrBadRequest)
	}
	return s.repo.DeleteSample(ctx, sampleID)
}

type ClientConfigOutput struct {
	EdgeAcceptThreshold    float64  `json:"edgeAcceptThreshold"`
	CloudFallbackThreshold float64  `json:"cloudFallbackThreshold"`
	CopyStyleVersion       string   `json:"copyStyleVersion"`
	ModelVersion           string   `json:"modelVersion"`
	ABBucket               int      `json:"abBucket"`
	ShareTemplates         []string `json:"shareTemplates"`
}

func (s *Service) ClientConfig(userID string) ClientConfigOutput {
	bucket := repository.ABBucket(userID, 3)
	return ClientConfigOutput{
		EdgeAcceptThreshold:    s.thresholds.EdgeAccept,
		CloudFallbackThreshold: s.thresholds.CloudFallback,
		CopyStyleVersion:       "v1",
		ModelVersion:           s.modelVersion,
		ABBucket:               bucket,
		ShareTemplates: []string{
			"我家主子刚发布了最新需求文档，速看！",
			"猫总监在线发话：铲屎官请立即执行。",
			"今天的猫语翻译已出炉，笑到打鸣。",
		},
	}
}

func (s *Service) GenerateCopy(ctx context.Context, result domain.InferenceResult) (domain.CopyBlock, error) {
	result.NormalizeTopK(3)
	if err := result.Validate(); err != nil {
		return domain.CopyBlock{}, fmt.Errorf("%w: invalid result", domain.ErrBadRequest)
	}
	return s.copyClient.Generate(ctx, result, "v1")
}

func normalizeSuffix(suffix string) string {
	if suffix == "" {
		return ".jpg"
	}
	if suffix[0] != '.' {
		return "." + suffix
	}
	return suffix
}
