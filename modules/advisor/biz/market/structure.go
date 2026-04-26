package market

import baseCandle "j_ai_trade/common"

// BOSRetest captures a recent break-of-structure on this TF: a closed
// bar whose CLOSE crossed the price of the most recent confirmed pivot,
// plus whether price has come back to retest the broken level. For a
// scalper, "retesting" (price wicked back to the level) and "confirmed"
// (after touch, a bar closed back in BOS direction) are the actionable
// states; "pending" is informational only.
type BOSRetest struct {
	Direction      string  // "up" (broke SH) | "down" (broke SL) | ""
	Level          float64 // broken pivot price
	BarsSinceBreak int     // 0 = break is on the latest closed bar
	State          string  // "pending" | "retesting" | "confirmed"
}

// DetectBOSRetest scans pivots newest-first and returns the freshest
// BOS within the last maxAge closed bars. retestTol = 0.3·ATR matches
// the same proximity tolerance used by at_support / at_resistance —
// keeps the prompt's "near a level" semantics consistent across flags.
func DetectBOSRetest(closed []baseCandle.BaseCandle, pivots []Pivot, atr float64, maxAge int) BOSRetest {
	if atr <= 0 || len(closed) < 3 || len(pivots) == 0 {
		return BOSRetest{}
	}
	const retestTol = 0.3
	tol := retestTol * atr
	n := len(closed)
	minBarIdx := n - maxAge
	if minBarIdx < 0 {
		minBarIdx = 0
	}

	timeIdx := make(map[int64]int, n)
	for i, c := range closed {
		timeIdx[c.OpenTime.UTC().UnixNano()] = i
	}

	var best BOSRetest
	bestAge := -1

	for k := len(pivots) - 1; k >= 0; k-- {
		p := pivots[k]
		pivotIdx, ok := timeIdx[p.Time.UnixNano()]
		if !ok || pivotIdx >= n-1 {
			continue
		}
		breakIdx := -1
		for j := pivotIdx + 1; j < n; j++ {
			if p.Type == "SH" && closed[j].Close > p.Price {
				breakIdx = j
				break
			}
			if p.Type == "SL" && closed[j].Close < p.Price {
				breakIdx = j
				break
			}
		}
		if breakIdx < 0 || breakIdx < minBarIdx {
			continue
		}
		age := n - 1 - breakIdx
		if bestAge >= 0 && age >= bestAge {
			continue
		}

		touched, confirmed := false, false
		for j := breakIdx + 1; j < n; j++ {
			b := closed[j]
			if p.Type == "SH" {
				if b.Low <= p.Price+tol {
					touched = true
				}
				if touched && b.Close > p.Price {
					confirmed = true
				}
			} else {
				if b.High >= p.Price-tol {
					touched = true
				}
				if touched && b.Close < p.Price {
					confirmed = true
				}
			}
		}
		// Latest bar wicking into the zone counts as a live retest even
		// if no full close-back has happened — that's the actionable
		// "right now" state for the scalper.
		if !touched {
			last := closed[n-1]
			if p.Type == "SH" && last.Low <= p.Price+tol {
				touched = true
			}
			if p.Type == "SL" && last.High >= p.Price-tol {
				touched = true
			}
		}

		state := "pending"
		switch {
		case confirmed:
			state = "confirmed"
		case touched:
			state = "retesting"
		}

		dir := "up"
		if p.Type == "SL" {
			dir = "down"
		}
		best = BOSRetest{Direction: dir, Level: p.Price, BarsSinceBreak: age, State: state}
		bestAge = age
		if age == 0 {
			break
		}
	}
	return best
}

// FVG (Fair Value Gap) is a 3-bar imbalance: a gap between bar[i-2] and
// bar[i] that bar[i-1] failed to fill. Bull FVGs (low[i] > high[i-2])
// are support zones; bear FVGs (high[i] < low[i-2]) are resistance.
// Scalping relevance: price returning into the zone often bounces, so
// "filling" (price currently inside the zone) is the entry trigger.
type FVG struct {
	Direction string  // "bull" | "bear" | ""
	Top       float64
	Bottom    float64
	Age       int    // bars since the gap formed (newest = 0)
	State     string // "open" | "filling"
}

// DetectRecentFVG returns the freshest unfilled (or currently filling)
// FVG within the last maxAge bars. A gap is dropped once any later bar
// fully crosses it. We prefer "filling" over merely "open" — active
// mitigation is the actionable scalp scenario.
func DetectRecentFVG(closed []baseCandle.BaseCandle, maxAge int) FVG {
	n := len(closed)
	if n < 3 || maxAge < 3 {
		return FVG{}
	}
	minIdx := n - maxAge
	if minIdx < 2 {
		minIdx = 2
	}

	var best FVG
	bestAge := -1

	for i := n - 1; i >= minIdx; i-- {
		prev := closed[i-2]
		curr := closed[i]
		var dir string
		var top, bot float64
		switch {
		case curr.Low > prev.High:
			dir, top, bot = "bull", curr.Low, prev.High
		case curr.High < prev.Low:
			dir, top, bot = "bear", prev.Low, curr.High
		default:
			continue
		}
		fullyFilled, filling := false, false
		for j := i + 1; j < n; j++ {
			b := closed[j]
			if dir == "bull" && b.Low <= bot {
				fullyFilled = true
				break
			}
			if dir == "bear" && b.High >= top {
				fullyFilled = true
				break
			}
			if dir == "bull" && b.Low < top && b.Low > bot {
				filling = true
			}
			if dir == "bear" && b.High > bot && b.High < top {
				filling = true
			}
		}
		if fullyFilled {
			continue
		}
		last := closed[n-1]
		if dir == "bull" && last.Low < top && last.Low > bot {
			filling = true
		}
		if dir == "bear" && last.High > bot && last.High < top {
			filling = true
		}
		age := n - 1 - i
		state := "open"
		if filling {
			state = "filling"
		}
		// Prefer filling over open even if the open one is fresher;
		// among the same state, keep the freshest.
		switch {
		case best.Direction == "":
			// first match — accept
		case best.State == "filling" && state != "filling":
			continue
		case best.State == state && age >= bestAge:
			continue
		}
		best = FVG{Direction: dir, Top: top, Bottom: bot, Age: age, State: state}
		bestAge = age
	}
	return best
}
