package scalping

import "math"

// calculateVWAP calculates Volume Weighted Average Price
func calculateVWAP(prices, volumes []float64) []float64 {
	vwap := make([]float64, len(prices))
	var cumulativePV float64
	var cumulativeVolume float64

	for i := 0; i < len(prices); i++ {
		typicalPrice := prices[i]
		cumulativePV += typicalPrice * volumes[i]
		cumulativeVolume += volumes[i]
		vwap[i] = cumulativePV / cumulativeVolume
	}

	return vwap
}

// findSupportLevels identifies support levels from a slice of lows
func findSupportLevels(lows []float64, lookback int) []float64 {
	if len(lows) < lookback {
		return nil
	}

	var supports []float64
	for i := lookback; i < len(lows)-1; i++ {
		if lows[i] < lows[i-1] && lows[i] < lows[i+1] {
			supports = append(supports, lows[i])
		}
	}
	return supports
}

// findResistanceLevels identifies resistance levels from a slice of highs
func findResistanceLevels(highs []float64, lookback int) []float64 {
	if len(highs) < lookback+2 {
		return nil
	}

	var resistances []float64
	for i := lookback; i < len(highs)-1; i++ {
		if highs[i] > highs[i-1] && highs[i] > highs[i+1] {
			resistances = append(resistances, highs[i])
		}
	}
	return resistances
}

// findNearestSupport finds the nearest support level below the current price
func findNearestSupport(price float64, supports []float64) float64 {
	if len(supports) == 0 {
		return price * 0.95 // Default 5% below price if no supports found
	}

	nearest := supports[0]
	minDist := math.Abs(price - supports[0])

	for _, support := range supports {
		if support > price {
			continue
		}
		dist := math.Abs(price - support)
		if dist < minDist {
			minDist = dist
			nearest = support
		}
	}

	return nearest
}

// findNearestResistance finds the nearest resistance level above the current price
func findNearestResistance(price float64, resistances []float64) float64 {
	if len(resistances) == 0 {
		return price * 1.05 // Default 5% above price if no resistances found
	}

	nearest := resistances[0]
	minDist := math.Abs(resistances[0] - price)

	for _, resistance := range resistances {
		if resistance < price {
			continue
		}
		dist := math.Abs(resistance - price)
		if dist < minDist {
			minDist = dist
			nearest = resistance
		}
	}

	return nearest
}
