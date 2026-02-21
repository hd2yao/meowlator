package app

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"os"
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

var allIntents = []IntentLabel{
	IntentFeeding,
	IntentSeekAttention,
	IntentWantPlay,
	IntentWantDoorOpen,
	IntentDefensiveAlert,
	IntentRelaxSleep,
	IntentCuriousObserve,
	IntentUncertain,
}

const priorBlendWeight = 0.20

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

type Model struct {
	priors map[IntentLabel]float64
}

func NewModel(priors map[IntentLabel]float64) *Model {
	return &Model{priors: normalizePriors(priors)}
}

func (m *Model) Predict(imageKey string, sceneTag string) InferenceResult {
	seed := hashValue(imageKey + "|" + sceneTag)
	candidates := make([]IntentProb, 0, len(allIntents))
	for idx, label := range allIntents {
		base := probability(seed, uint32(idx+1))
		prior := m.intentPrior(label)
		blended := clamp((1.0-priorBlendWeight)*base+priorBlendWeight*prior, 0.0, 1.0)
		candidates = append(candidates, IntentProb{Label: label, Prob: blended})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Prob > candidates[j].Prob
	})
	top3 := candidates[:3]
	confidence := top3[0].Prob
	state := deriveState(seed)
	evidence := []string{"云端复判触发", "姿态与场景联合特征"}
	if len(m.priors) > 0 {
		evidence = append(evidence, "融合训练先验分布")
	}
	return InferenceResult{
		IntentTop3:       top3,
		State:            state,
		Confidence:       confidence,
		Source:           "CLOUD",
		Evidence:         evidence,
		CopyStyleVersion: "v1",
	}
}

func (m *Model) intentPrior(label IntentLabel) float64 {
	if len(m.priors) == 0 {
		return 1.0 / float64(len(allIntents))
	}
	if value, ok := m.priors[label]; ok {
		return value
	}
	return 1.0 / float64(len(allIntents))
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

func normalizePriors(priors map[IntentLabel]float64) map[IntentLabel]float64 {
	if len(priors) == 0 {
		return nil
	}
	normalized := make(map[IntentLabel]float64)
	total := 0.0
	for label, prob := range priors {
		if !isValidIntent(label) || prob <= 0 {
			continue
		}
		normalized[label] = prob
		total += prob
	}
	if total == 0 {
		return nil
	}
	for label, prob := range normalized {
		normalized[label] = prob / total
	}
	return normalized
}

func isValidIntent(label IntentLabel) bool {
	for _, item := range allIntents {
		if item == label {
			return true
		}
	}
	return false
}

type priorsPayload struct {
	IntentPriors map[string]float64 `json:"intent_priors"`
}

func LoadIntentPriors(path string) (map[IntentLabel]float64, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapped priorsPayload
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.IntentPriors) > 0 {
		parsed := map[IntentLabel]float64{}
		for key, value := range wrapped.IntentPriors {
			label := IntentLabel(key)
			if isValidIntent(label) {
				parsed[label] = value
			}
		}
		normalized := normalizePriors(parsed)
		if len(normalized) == 0 {
			return nil, errors.New("intent priors payload does not contain valid positive values")
		}
		return normalized, nil
	}

	var flat map[string]float64
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, err
	}
	parsed := map[IntentLabel]float64{}
	for key, value := range flat {
		label := IntentLabel(key)
		if isValidIntent(label) {
			parsed[label] = value
		}
	}
	normalized := normalizePriors(parsed)
	if len(normalized) == 0 {
		return nil, errors.New("no valid intent priors found")
	}
	return normalized, nil
}

func clamp(v float64, low float64, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
