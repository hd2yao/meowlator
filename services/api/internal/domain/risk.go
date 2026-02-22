package domain

import (
	"fmt"
	"strings"
)

type PainRiskLevel string

const (
	PainRiskLow  PainRiskLevel = "LOW"
	PainRiskMid  PainRiskLevel = "MID"
	PainRiskHigh PainRiskLevel = "HIGH"
)

const PainRiskDisclaimer = "非医疗诊断，仅作风险提示；若持续异常请咨询兽医。"

type RiskInfo struct {
	PainRiskScore float64       `json:"painRiskScore"`
	PainRiskLevel PainRiskLevel `json:"painRiskLevel"`
	RiskEvidence  []string      `json:"riskEvidence"`
	Disclaimer    string        `json:"disclaimer"`
}

func (r *RiskInfo) Validate() error {
	if r == nil {
		return nil
	}
	if r.PainRiskScore < 0 || r.PainRiskScore > 1 {
		return fmt.Errorf("painRiskScore must be between 0 and 1")
	}
	switch r.PainRiskLevel {
	case PainRiskLow, PainRiskMid, PainRiskHigh:
	default:
		return fmt.Errorf("invalid painRiskLevel: %s", r.PainRiskLevel)
	}
	if strings.TrimSpace(r.Disclaimer) == "" {
		return fmt.Errorf("risk disclaimer is required")
	}
	return nil
}

func EvaluatePainRisk(result InferenceResult) *RiskInfo {
	score := 0.10
	evidence := []string{}

	switch result.State.Tension {
	case LevelHigh:
		score += 0.35
		evidence = append(evidence, "紧张度高")
	case LevelMid:
		score += 0.18
		evidence = append(evidence, "紧张度中等")
	}

	switch result.State.Comfort {
	case LevelLow:
		score += 0.35
		evidence = append(evidence, "舒适度低")
	case LevelMid:
		score += 0.15
		evidence = append(evidence, "舒适度中等")
	}

	switch result.State.Arousal {
	case LevelHigh:
		score += 0.15
		evidence = append(evidence, "兴奋度高")
	case LevelMid:
		score += 0.07
	}

	if len(result.IntentTop3) > 0 && result.IntentTop3[0].Label == IntentDefensiveAlert {
		score += 0.10
		evidence = append(evidence, "主意图为警戒防御")
	}

	if len(result.Evidence) > 0 {
		evidence = append(evidence, "结合视觉行为证据")
	}
	if len(evidence) == 0 {
		evidence = append(evidence, "状态信号不足")
	}

	finalScore := roundRiskScore(clampRiskScore(score))
	return &RiskInfo{
		PainRiskScore: finalScore,
		PainRiskLevel: classifyPainRisk(finalScore),
		RiskEvidence:  evidence,
		Disclaimer:    PainRiskDisclaimer,
	}
}

func classifyPainRisk(score float64) PainRiskLevel {
	switch {
	case score >= 0.75:
		return PainRiskHigh
	case score >= 0.50:
		return PainRiskMid
	default:
		return PainRiskLow
	}
}

func clampRiskScore(score float64) float64 {
	switch {
	case score < 0:
		return 0
	case score > 1:
		return 1
	default:
		return score
	}
}

func roundRiskScore(score float64) float64 {
	return float64(int(score*1000+0.5)) / 1000
}
