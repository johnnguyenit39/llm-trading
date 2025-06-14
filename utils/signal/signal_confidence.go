package signal

// getConfidenceIndicator returns a visual representation of confidence
func SetConfidenceIndicator(confidence float64) string {
	switch {
	case confidence >= 0.9:
		return "🟢🟢🟢"
	case confidence >= 0.8:
		return "🟢🟢🟡"
	case confidence >= 0.7:
		return "🟢🟡🟡"
	case confidence >= 0.6:
		return "🟡🟡🟡"
	default:
		return "🟡🟡🔴"
	}
}
