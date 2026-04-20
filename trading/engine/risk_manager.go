package engine

import (
	"math"

	baseCandle "j_ai_trade/common"
)

// RiskManager computes position size and leverage from ATR volatility and per-pair caps.
//
// Core rules:
//   - Risk a fixed fraction of equity per trade (default 1%).
//   - Position size = RiskUSD / SL distance → identical dollar risk across pairs.
//   - Leverage derived from notional / margin; capped per pair.
//   - Very volatile pairs (ATR > 5%) get an extra 0.7x safety reduction.
//   - When any cap forces a smaller notional, *actual* RiskUSD shrinks; we
//     report the true number and refuse the trade if it's below a configurable
//     floor relative to the intended risk.
type RiskManager struct {
	RiskPerTradePct     float64                // 0.01 = 1%
	MarginRatio         float64                // notional -> margin, e.g. 0.10 = 10x base
	MaxTotalNotional    float64                // total notional / equity, e.g. 3.0
	MinActualRiskRatio  float64                // 0.5 → fail if actual risk < 50% of intended
	PairConfigs         map[string]PairConfig

	// FeeRateTaker is the per-side taker fee (e.g. 0.0004 = 0.04%). Round-trip
	// cost is 2x this. Used to enforce a minimum SL distance so fees don't
	// dominate the risk budget.
	FeeRateTaker float64
}

// NewDefaultRiskManager returns a risk manager with sensible defaults.
func NewDefaultRiskManager() *RiskManager {
	return &RiskManager{
		RiskPerTradePct:    0.01,
		MarginRatio:        0.10,
		MaxTotalNotional:   3.0,
		MinActualRiskRatio: 0.5,
		PairConfigs:        DefaultPairConfigs(),
		FeeRateTaker:       0.0004,
	}
}

type SizeCalc struct {
	Notional        float64
	Leverage        float64
	Quantity        float64
	IntendedRiskUSD float64 // what the fixed-fractional rule asked for
	ActualRiskUSD   float64 // what the trade will actually lose on SL (after caps)
	SLDistPct       float64
	CappedBy        string // "" | "leverage" | "volatility" | "min_notional"
}

// CalculateSize computes qty / notional / leverage for a given trade setup.
// atrPct is ATR expressed as fraction of price (e.g. 0.015 = 1.5%).
// Returns ok=false with a reason when the trade cannot be sized within caps
// or when actual risk would fall below MinActualRiskRatio of intended.
func (r *RiskManager) CalculateSize(symbol string, equity, entryPrice, stopLoss, atrPct float64) (SizeCalc, bool, string) {
	if equity <= 0 || entryPrice <= 0 {
		return SizeCalc{}, false, "invalid equity or entry price"
	}

	slDistPct := math.Abs(entryPrice-stopLoss) / entryPrice
	// Floor SL distance at 3x round-trip fees so fees don't eat the entire
	// risk budget on super-tight stops.
	minSL := 6 * r.FeeRateTaker
	if minSL < 0.003 {
		minSL = 0.003
	}
	if slDistPct < minSL {
		return SizeCalc{}, false, "SL too tight vs fees"
	}

	intendedRisk := equity * r.RiskPerTradePct
	notional := intendedRisk / slDistPct
	qty := notional / entryPrice
	leverage := notional / (equity * r.MarginRatio)

	pair := LookupPairConfig(r.PairConfigs, symbol)
	cappedBy := ""

	if leverage > pair.MaxLeverage {
		leverage = pair.MaxLeverage
		notional = equity * r.MarginRatio * leverage
		qty = notional / entryPrice
		cappedBy = "leverage"
	}

	if atrPct > 0.05 {
		qty *= 0.7
		notional *= 0.7
		if cappedBy == "" {
			cappedBy = "volatility"
		}
	}

	if notional < pair.MinNotional {
		return SizeCalc{}, false, "below min notional"
	}

	// Actual risk is the dollar loss if SL hits.
	actualRisk := notional * slDistPct

	if actualRisk < intendedRisk*r.MinActualRiskRatio {
		return SizeCalc{}, false, "caps shrink risk below floor"
	}

	leverage = math.Round(leverage)
	if leverage < 1 {
		leverage = 1
	}

	return SizeCalc{
		Notional:        notional,
		Leverage:        leverage,
		Quantity:        qty,
		IntendedRiskUSD: intendedRisk,
		ActualRiskUSD:   actualRisk,
		SLDistPct:       slDistPct,
		CappedBy:        cappedBy,
	}, true, ""
}

// NetRRAfterFees returns the reward:risk ratio *after* subtracting round-trip
// taker fees from both the reward and the risk legs.
//
// Assumptions: entry and exit are both taker (worst case). Fees scale with
// notional, so expressing them in price terms = entry * 2 * feeRate for a
// round trip.
func (r *RiskManager) NetRRAfterFees(entry, sl, tp float64) float64 {
	if entry <= 0 {
		return 0
	}
	feeCost := entry * 2 * r.FeeRateTaker
	reward := math.Abs(tp-entry) - feeCost
	risk := math.Abs(entry-sl) + feeCost
	if risk <= 0 {
		return 0
	}
	if reward <= 0 {
		return 0
	}
	return reward / risk
}

// ATRPercent computes ATR as a fraction of the latest close.
// Returns 0 if not enough candles.
func ATRPercent(candles []baseCandle.BaseCandle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}
	atr := 0.0
	for i := len(candles) - period; i < len(candles); i++ {
		h := candles[i].High
		l := candles[i].Low
		pc := candles[i-1].Close
		tr := h - l
		if v := math.Abs(h - pc); v > tr {
			tr = v
		}
		if v := math.Abs(l - pc); v > tr {
			tr = v
		}
		atr += tr
	}
	atr /= float64(period)
	close := candles[len(candles)-1].Close
	if close == 0 {
		return 0
	}
	return atr / close
}
