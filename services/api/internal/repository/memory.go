package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type MemoryRepository struct {
	mu             sync.RWMutex
	samples        map[string]*domain.Sample
	feedback       map[string]*domain.Feedback
	feedbackByUser map[string][]*domain.Feedback
	sessions       map[string]*domain.UserSession
	models         map[string]*domain.ModelRegistry
	riskEvents     map[string][]*domain.RiskInfo
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		samples:        make(map[string]*domain.Sample),
		feedback:       make(map[string]*domain.Feedback),
		feedbackByUser: make(map[string][]*domain.Feedback),
		sessions:       make(map[string]*domain.UserSession),
		models:         make(map[string]*domain.ModelRegistry),
		riskEvents:     make(map[string][]*domain.RiskInfo),
	}
}

func (r *MemoryRepository) CreateSample(ctx context.Context, sample *domain.Sample) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.samples[sample.SampleID]; exists {
		return domain.ErrConflict
	}
	copied := *sample
	r.samples[sample.SampleID] = &copied
	return nil
}

func (r *MemoryRepository) GetSample(ctx context.Context, sampleID string) (*domain.Sample, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	sample, ok := r.samples[sampleID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copied := *sample
	return &copied, nil
}

func (r *MemoryRepository) UpdatePredictions(ctx context.Context, sampleID string, edgePred, finalPred *domain.InferenceResult, sceneTag string, modelVersion string) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	sample, ok := r.samples[sampleID]
	if !ok {
		return domain.ErrNotFound
	}
	if edgePred != nil {
		edgeCopy := *edgePred
		sample.EdgePred = &edgeCopy
	}
	if finalPred != nil {
		finalCopy := *finalPred
		sample.FinalPred = &finalCopy
	}
	if sceneTag != "" {
		sample.SceneTag = sceneTag
	}
	if modelVersion != "" {
		sample.ModelVersion = modelVersion
	}
	return nil
}

func (r *MemoryRepository) SaveFeedback(ctx context.Context, fb *domain.Feedback) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.samples[fb.SampleID]; !ok {
		return domain.ErrNotFound
	}
	if _, exists := r.feedback[fb.FeedbackID]; exists {
		return domain.ErrConflict
	}
	copyFB := *fb
	r.feedback[fb.FeedbackID] = &copyFB
	r.feedbackByUser[fb.UserID] = append(r.feedbackByUser[fb.UserID], &copyFB)
	return nil
}

func (r *MemoryRepository) DeleteSample(ctx context.Context, sampleID string) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.samples[sampleID]; !ok {
		return domain.ErrNotFound
	}
	delete(r.samples, sampleID)
	for id, fb := range r.feedback {
		if fb.SampleID == sampleID {
			delete(r.feedback, id)
		}
	}
	for userID, list := range r.feedbackByUser {
		filtered := list[:0]
		for _, fb := range list {
			if fb.SampleID != sampleID {
				filtered = append(filtered, fb)
			}
		}
		r.feedbackByUser[userID] = filtered
	}
	return nil
}

func (r *MemoryRepository) UserFeedbackStats(ctx context.Context, userID string) (total int, extremeRatio float64, suspicious bool) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := r.feedbackByUser[userID]
	total = len(list)
	if total == 0 {
		return 0, 0, false
	}
	extreme := 0
	conflicts := 0
	for _, fb := range list {
		if fb.TrueLabel == domain.IntentDefensiveAlert || fb.TrueLabel == domain.IntentUncertain {
			extreme++
		}
		if !fb.IsCorrect {
			conflicts++
		}
	}
	extremeRatio = float64(extreme) / float64(total)
	suspicious = total >= 3 && conflicts == total
	return total, extremeRatio, suspicious
}

func (r *MemoryRepository) DeleteExpiredSamples(ctx context.Context, expireBefore int64) (int, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Unix(expireBefore, 0)
	for token, session := range r.sessions {
		if !session.ExpiresAt.After(now) {
			delete(r.sessions, token)
		}
	}
	deleted := 0
	for sampleID, sample := range r.samples {
		if sample.ExpireAt <= expireBefore {
			delete(r.samples, sampleID)
			deleted++
		}
	}
	if deleted == 0 {
		return 0, nil
	}
	for id, fb := range r.feedback {
		if _, ok := r.samples[fb.SampleID]; !ok {
			delete(r.feedback, id)
		}
	}
	for userID, list := range r.feedbackByUser {
		filtered := list[:0]
		for _, fb := range list {
			if _, ok := r.samples[fb.SampleID]; ok {
				filtered = append(filtered, fb)
			}
		}
		r.feedbackByUser[userID] = filtered
	}
	return deleted, nil
}

func (r *MemoryRepository) CreateSession(ctx context.Context, session *domain.UserSession) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	copySession := *session
	r.sessions[session.SessionToken] = &copySession
	return nil
}

func (r *MemoryRepository) GetSession(ctx context.Context, token string) (*domain.UserSession, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[token]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copySession := *session
	return &copySession, nil
}

func (r *MemoryRepository) UpsertModelRegistry(ctx context.Context, model domain.ModelRegistry) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	copyModel := model
	r.models[model.ModelVersion] = &copyModel
	return nil
}

func (r *MemoryRepository) UpdateModelStatus(ctx context.Context, modelVersion string, status domain.ModelStatus, rolloutRatio float64, targetBucket int) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	model, ok := r.models[modelVersion]
	if !ok {
		model = &domain.ModelRegistry{
			ModelVersion: modelVersion,
			TaskScope:    "intent_state_risk",
			CreatedAt:    time.Now(),
		}
		r.models[modelVersion] = model
	}
	if status == domain.ModelStatusActive {
		for version, item := range r.models {
			if version == modelVersion {
				continue
			}
			if item.Status == domain.ModelStatusActive || item.Status == domain.ModelStatusGray {
				item.Status = domain.ModelStatusRolledBack
			}
		}
	}
	model.Status = status
	model.RolloutRatio = rolloutRatio
	model.TargetBucket = targetBucket
	return nil
}

func (r *MemoryRepository) GetActiveModel(ctx context.Context) (*domain.ModelRegistry, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, model := range r.models {
		if model.Status == domain.ModelStatusActive || model.Status == domain.ModelStatusGray {
			copyModel := *model
			return &copyModel, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *MemoryRepository) SaveRiskEvent(ctx context.Context, sampleID string, risk *domain.RiskInfo) error {
	_ = ctx
	if risk == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	copyRisk := *risk
	r.riskEvents[sampleID] = append(r.riskEvents[sampleID], &copyRisk)
	return nil
}

func GenerateID(prefix string) string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("%s_%d", prefix, now)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(raw))
}

func ABBucket(userID string, buckets int) int {
	if buckets <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	return int(h.Sum32() % uint32(buckets))
}
