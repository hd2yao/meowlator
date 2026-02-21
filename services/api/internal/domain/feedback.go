package domain

const (
	WeightCorrected = 1.0
	WeightConfirmed = 0.6
	WeightModelOnly = 0.2
)

func FeedbackWeight(isCorrect bool) float64 {
	if isCorrect {
		return WeightConfirmed
	}
	return WeightCorrected
}

func ReliabilityScore(totalFeedback int, extremeRatio float64, suspicious bool) float64 {
	score := 1.0
	if totalFeedback > 30 {
		score += 0.1
	}
	if extremeRatio > 0.8 {
		score -= 0.3
	}
	if suspicious {
		score -= 0.4
	}
	if score < 0.1 {
		return 0.1
	}
	if score > 1 {
		return 1
	}
	return score
}
