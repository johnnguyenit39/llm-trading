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

