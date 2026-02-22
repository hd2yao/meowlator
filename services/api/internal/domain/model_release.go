package domain

import "time"

type ModelStatus string

const (
	ModelStatusCandidate  ModelStatus = "CANDIDATE"
	ModelStatusGray       ModelStatus = "GRAY"
	ModelStatusActive     ModelStatus = "ACTIVE"
	ModelStatusRolledBack ModelStatus = "ROLLED_BACK"
)

func (s ModelStatus) IsValid() bool {
	switch s {
	case ModelStatusCandidate, ModelStatusGray, ModelStatusActive, ModelStatusRolledBack:
		return true
	default:
		return false
	}
}

type ModelRegistry struct {
	ModelVersion string
	TaskScope    string
	MetricsJSON  string
	Status       ModelStatus
	RolloutRatio float64
	TargetBucket int
	CreatedAt    time.Time
}

type UserSession struct {
	SessionToken string
	UserID       string
	WechatCode   string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
