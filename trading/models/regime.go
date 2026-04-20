package models

// Regime is a coarse classification of current market conditions.
// It is used by the ensemble to decide which strategies are *eligible*
// to contribute to a consensus at this point in time.
type Regime string

const (
	RegimeTrendUp   Regime = "TREND_UP"
	RegimeTrendDown Regime = "TREND_DOWN"
	RegimeRange     Regime = "RANGE"
	RegimeChoppy    Regime = "CHOPPY" // no clear trend, no clean range
)

// IsTrend reports whether the regime is a directional trend.
func (r Regime) IsTrend() bool {
	return r == RegimeTrendUp || r == RegimeTrendDown
}
