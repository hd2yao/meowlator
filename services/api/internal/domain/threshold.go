package domain

type ThresholdConfig struct {
	EdgeAccept    float64
	CloudFallback float64
}

type ThresholdDecision struct {
	UseEdge       bool
	NeedCloud     bool
	ForceFeedback bool
}

func DecideThreshold(edgeConfidence float64, deviceCapable bool, cfg ThresholdConfig) ThresholdDecision {
	if !deviceCapable {
		return ThresholdDecision{NeedCloud: true, ForceFeedback: true}
	}
	if edgeConfidence >= cfg.EdgeAccept {
		return ThresholdDecision{UseEdge: true}
	}
	if edgeConfidence < cfg.CloudFallback {
		return ThresholdDecision{NeedCloud: true, ForceFeedback: true}
	}
	return ThresholdDecision{NeedCloud: true}
}
