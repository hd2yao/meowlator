package domain

import (
	"errors"
	"fmt"
	"sort"
)

type IntentLabel string

const (
	IntentFeeding        IntentLabel = "FEEDING"
	IntentSeekAttention  IntentLabel = "SEEK_ATTENTION"
	IntentWantPlay       IntentLabel = "WANT_PLAY"
	IntentWantDoorOpen   IntentLabel = "WANT_DOOR_OPEN"
	IntentDefensiveAlert IntentLabel = "DEFENSIVE_ALERT"
	IntentRelaxSleep     IntentLabel = "RELAX_SLEEP"
	IntentCuriousObserve IntentLabel = "CURIOUS_OBSERVE"
	IntentUncertain      IntentLabel = "UNCERTAIN"
)

var validIntentLabels = map[IntentLabel]struct{}{
	IntentFeeding:        {},
	IntentSeekAttention:  {},
	IntentWantPlay:       {},
	IntentWantDoorOpen:   {},
	IntentDefensiveAlert: {},
	IntentRelaxSleep:     {},
	IntentCuriousObserve: {},
	IntentUncertain:      {},
}

func ParseIntentLabel(input string) (IntentLabel, error) {
	label := IntentLabel(input)
	if _, ok := validIntentLabels[label]; !ok {
		return "", fmt.Errorf("invalid intent label: %s", input)
	}
	return label, nil
}

type Level3 string

const (
	LevelLow  Level3 = "LOW"
	LevelMid  Level3 = "MID"
	LevelHigh Level3 = "HIGH"
)

func ParseLevel3(input string) (Level3, error) {
	level := Level3(input)
	switch level {
	case LevelLow, LevelMid, LevelHigh:
		return level, nil
	default:
		return "", fmt.Errorf("invalid level: %s", input)
	}
}

type State3D struct {
	Tension Level3 `json:"tension"`
	Arousal Level3 `json:"arousal"`
	Comfort Level3 `json:"comfort"`
}

func (s State3D) Validate() error {
	if _, err := ParseLevel3(string(s.Tension)); err != nil {
		return err
	}
	if _, err := ParseLevel3(string(s.Arousal)); err != nil {
		return err
	}
	if _, err := ParseLevel3(string(s.Comfort)); err != nil {
		return err
	}
	return nil
}

type IntentProb struct {
	Label IntentLabel `json:"label"`
	Prob  float64     `json:"prob"`
}

type InferenceResult struct {
	IntentTop3       []IntentProb `json:"intentTop3"`
	State            State3D      `json:"state"`
	Confidence       float64      `json:"confidence"`
	Source           string       `json:"source"`
	Evidence         []string     `json:"evidence"`
	CopyStyleVersion string       `json:"copyStyleVersion"`
}

func (r *InferenceResult) Validate() error {
	if len(r.IntentTop3) == 0 {
		return errors.New("intentTop3 cannot be empty")
	}
	for _, item := range r.IntentTop3 {
		if _, err := ParseIntentLabel(string(item.Label)); err != nil {
			return err
		}
		if item.Prob < 0 || item.Prob > 1 {
			return fmt.Errorf("invalid prob for %s", item.Label)
		}
	}
	if err := r.State.Validate(); err != nil {
		return err
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return errors.New("confidence must be between 0 and 1")
	}
	if r.Source != "EDGE" && r.Source != "CLOUD" {
		return errors.New("source must be EDGE or CLOUD")
	}
	return nil
}

func (r *InferenceResult) NormalizeTopK(k int) {
	sort.Slice(r.IntentTop3, func(i, j int) bool {
		return r.IntentTop3[i].Prob > r.IntentTop3[j].Prob
	})
	if k > 0 && len(r.IntentTop3) > k {
		r.IntentTop3 = r.IntentTop3[:k]
	}
	if len(r.IntentTop3) > 0 {
		r.Confidence = r.IntentTop3[0].Prob
	}
}

type CopyBlock struct {
	CatLine    string `json:"catLine"`
	Evidence   string `json:"evidence"`
	ShareTitle string `json:"shareTitle"`
}

type Sample struct {
	SampleID     string           `json:"sampleId"`
	UserID       string           `json:"userId"`
	CatID        string           `json:"catId"`
	ImageKey     string           `json:"imageKey"`
	SceneTag     string           `json:"sceneTag"`
	ModelVersion string           `json:"modelVersion"`
	EdgePred     *InferenceResult `json:"edgePred,omitempty"`
	FinalPred    *InferenceResult `json:"finalPred,omitempty"`
	CreatedAt    int64            `json:"createdAt"`
	ExpireAt     int64            `json:"expireAt"`
}

type Feedback struct {
	FeedbackID       string      `json:"feedbackId"`
	SampleID         string      `json:"sampleId"`
	UserID           string      `json:"userId"`
	IsCorrect        bool        `json:"isCorrect"`
	TrueLabel        IntentLabel `json:"trueLabel"`
	ReliabilityScore float64     `json:"reliabilityScore"`
	TrainingWeight   float64     `json:"trainingWeight"`
	CreatedAt        int64       `json:"createdAt"`
}
