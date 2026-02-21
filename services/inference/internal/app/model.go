package app

import (
	"hash/fnv"
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

type Level3 string

const (
	LevelLow  Level3 = "LOW"
	LevelMid  Level3 = "MID"
	LevelHigh Level3 = "HIGH"
)

type State3D struct {
	Tension Level3 `json:"tension"`
	Arousal Level3 `json:"arousal"`
	Comfort Level3 `json:"comfort"`
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

type Model struct{}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) Predict(imageKey string, sceneTag string) InferenceResult {
	_ = m
	seed := hashValue(imageKey + "|" + sceneTag)
	candidates := []IntentProb{
		{Label: IntentFeeding, Prob: probability(seed, 1)},
		{Label: IntentSeekAttention, Prob: probability(seed, 2)},
		{Label: IntentWantPlay, Prob: probability(seed, 3)},
		{Label: IntentWantDoorOpen, Prob: probability(seed, 4)},
		{Label: IntentDefensiveAlert, Prob: probability(seed, 5)},
		{Label: IntentRelaxSleep, Prob: probability(seed, 6)},
		{Label: IntentCuriousObserve, Prob: probability(seed, 7)},
		{Label: IntentUncertain, Prob: probability(seed, 8)},
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Prob > candidates[j].Prob
	})
	top3 := candidates[:3]
	confidence := top3[0].Prob
	state := deriveState(seed)
	return InferenceResult{
		IntentTop3:       top3,
		State:            state,
		Confidence:       confidence,
		Source:           "CLOUD",
		Evidence:         []string{"云端复判触发", "姿态与场景联合特征"},
		CopyStyleVersion: "v1",
	}
}

func probability(seed uint32, salt uint32) float64 {
	mixed := (seed ^ (salt * 2654435761)) % 100
	return 0.30 + float64(mixed)/200
}

func deriveState(seed uint32) State3D {
	return State3D{
		Tension: bucket(seed % 3),
		Arousal: bucket((seed / 3) % 3),
		Comfort: bucket((seed / 7) % 3),
	}
}

func bucket(v uint32) Level3 {
	switch v {
	case 0:
		return LevelLow
	case 1:
		return LevelMid
	default:
		return LevelHigh
	}
}

func hashValue(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}
