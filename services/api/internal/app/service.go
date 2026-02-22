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
	DeleteExpiredSamples(ctx context.Context, expireBefore int64) (int, error)
	CreateSession(ctx context.Context, session *domain.UserSession) error
	GetSession(ctx context.Context, token string) (*domain.UserSession, error)
	UpsertModelRegistry(ctx context.Context, model domain.ModelRegistry) error
	UpdateModelStatus(ctx context.Context, modelVersion string, status domain.ModelStatus, rolloutRatio float64, targetBucket int) error
	GetActiveModel(ctx context.Context) (*domain.ModelRegistry, error)
	SaveRiskEvent(ctx context.Context, sampleID string, risk *domain.RiskInfo) error
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
	edgeWhitelist []string
}

func NewService(repo Repository, inference InferenceClient, copyClient CopyClient, thresholds Thresholds, retentionDays int, modelVersion string, painRisk bool, edgeWhitelist []string) *Service {
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
		edgeWhitelist: append([]string{}, edgeWhitelist...),
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
			ModelHash:      in.EdgeRuntime.ModelHash,
			InputShape:     in.EdgeRuntime.InputShape,
			LoadMS:         in.EdgeRuntime.LoadMS,
			InferMS:        in.EdgeRuntime.InferMS,
			DeviceModel:    in.EdgeRuntime.DeviceModel,
			FailureCode:    in.EdgeRuntime.FailureCode,
			FailureReason:  in.EdgeRuntime.FailureReason,
			FallbackUsed:   fallbackUsed,
			UsedEdgeResult: finalResult.Source == "EDGE",
		}
	}
	if s.painRisk {
		finalResult.Risk = domain.EvaluatePainRisk(finalResult)
		_ = s.repo.SaveRiskEvent(ctx, in.SampleID, finalResult.Risk)
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
	EdgeDeviceWhitelist    []string `json:"edgeDeviceWhitelist"`
	ModelRollout           struct {
		ActiveModel  string  `json:"activeModel"`
		RolloutRatio float64 `json:"rolloutRatio"`
		TargetBucket int     `json:"targetBucket"`
	} `json:"modelRollout"`
	RiskEnabled   bool `json:"riskEnabled"`
	ABBucketRules struct {
		TotalBuckets int `json:"totalBuckets"`
	} `json:"abBucketRules"`
}

func (s *Service) ClientConfig(userID string) ClientConfigOutput {
	bucket := repository.ABBucket(userID, 3)
	out := ClientConfigOutput{
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
		EdgeDeviceWhitelist: append([]string{}, s.edgeWhitelist...),
		RiskEnabled:         s.painRisk,
	}
	out.ABBucketRules.TotalBuckets = 3
	active, err := s.repo.GetActiveModel(context.Background())
	if err == nil && active != nil {
		out.ModelRollout.ActiveModel = active.ModelVersion
		out.ModelRollout.RolloutRatio = active.RolloutRatio
		out.ModelRollout.TargetBucket = active.TargetBucket
	} else {
		out.ModelRollout.ActiveModel = s.modelVersion
		out.ModelRollout.RolloutRatio = 1.0
		out.ModelRollout.TargetBucket = 0
	}
	return out
}

type LoginInput struct {
	Code string
}

type LoginOutput struct {
	UserID       string `json:"userId"`
	SessionToken string `json:"sessionToken"`
	ExpiresAt    int64  `json:"expiresAt"`
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	if in.Code == "" {
		return nil, fmt.Errorf("%w: code is required", domain.ErrBadRequest)
	}
	userID := fmt.Sprintf("user_%x", repository.ABBucket(in.Code, 1<<20))
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	session := &domain.UserSession{
		SessionToken: repository.GenerateID("sess"),
		UserID:       userID,
		WechatCode:   in.Code,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &LoginOutput{
		UserID:       userID,
		SessionToken: session.SessionToken,
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}

func (s *Service) ValidateSession(ctx context.Context, userID string, token string) error {
	if userID == "" || token == "" {
		return fmt.Errorf("%w: missing user session", domain.ErrUnauthorized)
	}
	session, err := s.repo.GetSession(ctx, token)
	if err != nil {
		return fmt.Errorf("%w: invalid session", domain.ErrUnauthorized)
	}
	if session.UserID != userID {
		return fmt.Errorf("%w: user mismatch", domain.ErrUnauthorized)
	}
	if time.Now().After(session.ExpiresAt) {
		return fmt.Errorf("%w: session expired", domain.ErrUnauthorized)
	}
	return nil
}

func (s *Service) CleanupExpiredSamples(ctx context.Context) (int, error) {
	return s.repo.DeleteExpiredSamples(ctx, time.Now().Unix())
}

func (s *Service) RolloutModel(ctx context.Context, modelVersion string, rolloutRatio float64, targetBucket int) error {
	if modelVersion == "" {
		return fmt.Errorf("%w: modelVersion is required", domain.ErrBadRequest)
	}
	if rolloutRatio < 0 || rolloutRatio > 1 {
		return fmt.Errorf("%w: rolloutRatio must be between 0 and 1", domain.ErrBadRequest)
	}
	if targetBucket < 0 {
		return fmt.Errorf("%w: targetBucket must be >= 0", domain.ErrBadRequest)
	}
	return s.repo.UpdateModelStatus(ctx, modelVersion, domain.ModelStatusGray, rolloutRatio, targetBucket)
}

func (s *Service) ActivateModel(ctx context.Context, modelVersion string) error {
	if modelVersion == "" {
		return fmt.Errorf("%w: modelVersion is required", domain.ErrBadRequest)
	}
	return s.repo.UpdateModelStatus(ctx, modelVersion, domain.ModelStatusActive, 1.0, 0)
}

func (s *Service) RegisterModelEvaluation(ctx context.Context, modelVersion string, metricsJSON string) error {
	if modelVersion == "" || metricsJSON == "" {
		return fmt.Errorf("%w: modelVersion and metrics are required", domain.ErrBadRequest)
	}
	return s.repo.UpsertModelRegistry(ctx, domain.ModelRegistry{
		ModelVersion: modelVersion,
		TaskScope:    "intent_state_risk",
		MetricsJSON:  metricsJSON,
		Status:       domain.ModelStatusCandidate,
		RolloutRatio: 0,
		TargetBucket: 0,
		CreatedAt:    time.Now(),
	})
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
