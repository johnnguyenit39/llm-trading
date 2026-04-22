package market

import (
	"math"
	"time"

	baseCandle "j_ai_trade/common"
)

// Pivot is a confirmed swing high (SH) or swing low (SL) with an
// HH/HL/LH/LL label relative to the previous same-type pivot in the
// scanned window. Empty Label = first-of-type seen.
//
// Emitting a pivot SEQUENCE is the canonical way to let the LLM reason
// about subjective chart patterns (triangles, wedges, H&S) without
// hard-coding their detection — the LLM reads LH→HL→LH→HL as
// "converging range" on its own, no code heuristics needed.
type Pivot struct {
	Time  time.Time
	Price float64
	Type  string // "SH" | "SL"
	Label string // "HH" | "LH" | "HL" | "LL" | "EH" | "EL" | ""
}

// RecentPivots scans `candles` chronologically and returns up to
// `limit` most-recent confirmed pivots, each tagged with an HH/HL/LH/LL
// label relative to the prior same-type pivot. leftRight controls
// confirmation distance — 3 matches indicators.SwingHighLow and keeps
// us at the same "swing resolution" as the existing digest.
func RecentPivots(candles []baseCandle.BaseCandle, leftRight, limit int) []Pivot {
	n := len(candles)
	if n < leftRight*2+1 || limit <= 0 {
		return nil
	}
	var all []Pivot
	for i := leftRight; i < n-leftRight; i++ {
		if isPivotHigh(candles, i, leftRight) {
			all = append(all, Pivot{
				Time:  candles[i].OpenTime.UTC(),
				Price: candles[i].High,
				Type:  "SH",
			})
		}
		if isPivotLow(candles, i, leftRight) {
			all = append(all, Pivot{
				Time:  candles[i].OpenTime.UTC(),
				Price: candles[i].Low,
				Type:  "SL",
			})
		}
	}
	var lastSH, lastSL *Pivot
	for k := range all {
		p := &all[k]
		switch p.Type {
		case "SH":
			if lastSH != nil {
				switch {
				case p.Price > lastSH.Price:
					p.Label = "HH"
				case p.Price < lastSH.Price:
					p.Label = "LH"
				default:
					p.Label = "EH"
				}
			}
			lastSH = p
		case "SL":
			if lastSL != nil {
				switch {
				case p.Price > lastSL.Price:
					p.Label = "HL"
				case p.Price < lastSL.Price:
					p.Label = "LL"
				default:
					p.Label = "EL"
				}
			}
			lastSL = p
		}
	}
	if len(all) <= limit {
		return all
	}
	return all[len(all)-limit:]
}

func isPivotHigh(candles []baseCandle.BaseCandle, i, k int) bool {
	h := candles[i].High
	for j := i - k; j <= i+k; j++ {
		if j == i {
			continue
		}
		if candles[j].High >= h {
			return false
		}
	}
	return true
}

func isPivotLow(candles []baseCandle.BaseCandle, i, k int) bool {
	l := candles[i].Low
	for j := i - k; j <= i+k; j++ {
		if j == i {
			continue
		}
		if candles[j].Low <= l {
			return false
		}
	}
	return true
}

// DoubleStructure flags a double top / double bottom when the last two
// same-type pivots match in price (within tolerance·ATR) AND a pivot
// of the opposite type sits between them (the "valley" / "peak"). The
// LLM uses Level as the breakout / breakdown line.
type DoubleStructure struct {
	Kind  string // "double_top" | "double_bottom" | ""
	Level float64
}

// DetectDoubleTopBottom checks the last two same-type pivots only.
// tolerance is ATR fraction (0.3 = ±0.3 ATR means "same price level").
func DetectDoubleTopBottom(pivots []Pivot, atr, tolerance float64) DoubleStructure {
	if len(pivots) < 3 || atr <= 0 {
		return DoubleStructure{}
	}
	var shIdx, slIdx []int
	for i, p := range pivots {
		switch p.Type {
		case "SH":
			shIdx = append(shIdx, i)
		case "SL":
			slIdx = append(slIdx, i)
		}
	}
	if len(shIdx) >= 2 {
		a, b := shIdx[len(shIdx)-2], shIdx[len(shIdx)-1]
		if math.Abs(pivots[a].Price-pivots[b].Price) <= tolerance*atr {
			for _, si := range slIdx {
				if si > a && si < b {
					return DoubleStructure{
						Kind:  "double_top",
						Level: (pivots[a].Price + pivots[b].Price) / 2,
					}
				}
			}
		}
	}
	if len(slIdx) >= 2 {
		a, b := slIdx[len(slIdx)-2], slIdx[len(slIdx)-1]
		if math.Abs(pivots[a].Price-pivots[b].Price) <= tolerance*atr {
			for _, si := range shIdx {
				if si > a && si < b {
					return DoubleStructure{
						Kind:  "double_bottom",
						Level: (pivots[a].Price + pivots[b].Price) / 2,
					}
				}
			}
		}
	}
	return DoubleStructure{}
}

// RangeStructure describes a horizontal trading range detected over a
// rolling window: repeated touches on both boundaries AND narrow total
// width. "Narrow" = under widthMax·ATR so we don't call every swing a
// rectangle. Only IsRange=true results should drive a mean-reversion
// strategy; the other fields are diagnostic.
type RangeStructure struct {
	IsRange    bool
	Top        float64
	Bottom     float64
	TopTouches int
	BotTouches int
	WidthATR   float64
}

// DetectRange runs a minimal rectangle test over the last `window`
// closed bars: count touches near the window's high and low (±0.3 ATR),
// and declare a range only if both sides reach minTouches=3 and the
// total width stays under 4·ATR. Thresholds are conservative on purpose
// — false positives on "range" cause scalp buy-the-dip setups at real
// trend bottoms, which is expensive.
func DetectRange(candles []baseCandle.BaseCandle, atr float64, window int) RangeStructure {
	const (
		touchTol   = 0.3
		minTouches = 3
		widthMax   = 4.0
	)
	if len(candles) < window || atr <= 0 {
		return RangeStructure{}
	}
	slice := candles[len(candles)-window:]
	top, bot := slice[0].High, slice[0].Low
	for _, c := range slice {
		if c.High > top {
			top = c.High
		}
		if c.Low < bot {
			bot = c.Low
		}
	}
	width := (top - bot) / atr
	rs := RangeStructure{Top: top, Bottom: bot, WidthATR: width}
	if width > widthMax {
		return rs
	}
	tol := touchTol * atr
	for _, c := range slice {
		if c.High >= top-tol {
			rs.TopTouches++
		}
		if c.Low <= bot+tol {
			rs.BotTouches++
		}
	}
	if rs.TopTouches >= minTouches && rs.BotTouches >= minTouches {
		rs.IsRange = true
	}
	return rs
}
