package engine

import (
	"math"

	baseCandle "j_ai_trade/common"
)

// RiskManager computes position size and leverage from ATR volatility and per-pair caps.
// Rules of thumb:
//   - Risk a fixed fraction of equity per trade (default 1%).
//   - Position size scales inversely with SL distance → identical $ risk across pairs.
//   - Leverage derived from notional / margin; capped per pair.
//   - Very volatile pairs (ATR > 5%) get an extra size reduction as safety.
type RiskManager struct {
	RiskPerTradePct  float64                // 0.01 = 1%
	MarginRatio      float64                // notional -> margin, e.g. 0.10 = 10x base
	MaxTotalNotional float64                // total notional / equity, e.g. 3.0
	PairConfigs      map[string]PairConfig
}

// NewDefaultRiskManager returns a risk manager with sensible defaults.
func NewDefaultRiskManager() *RiskManager {
	return &RiskManager{
		RiskPerTradePct:  0.01,
		MarginRatio:      0.10,
		MaxTotalNotional: 3.0,
		PairConfigs:      DefaultPairConfigs(),
	}
}

type SizeCalc struct {
	Notional  float64
	Leverage  float64
	Quantity  float64
	RiskUSD   float64
	SLDistPct float64
}

// CalculateSize computes qty / notional / leverage for a given trade setup.
// atrPct is ATR expressed as fraction of price (e.g. 0.015 = 1.5%).
// Returns zero-valued SizeCalc + ok=false if inputs are invalid.
func (r *RiskManager) CalculateSize(symbol string, equity, entryPrice, stopLoss, atrPct float64) (SizeCalc, bool) {
	if equity <= 0 || entryPrice <= 0 {
		return SizeCalc{}, false
	}

	slDistPct := math.Abs(entryPrice-stopLoss) / entryPrice
	if slDistPct < 0.003 {
		slDistPct = 0.003 // floor 0.3% to prevent absurd sizes on very tight SL
	}

	riskUSD := equity * r.RiskPerTradePct
	notional := riskUSD / slDistPct
	qty := notional / entryPrice
	leverage := notional / (equity * r.MarginRatio)

	pair := LookupPairConfig(r.PairConfigs, symbol)
	if leverage > pair.MaxLeverage {
		leverage = pair.MaxLeverage
		notional = equity * r.MarginRatio * leverage
		qty = notional / entryPrice
	}

	// High-vol safety: if ATR > 5%, shrink size 0.7x
	if atrPct > 0.05 {
		qty *= 0.7
		notional *= 0.7
	}

	if notional < pair.MinNotional {
		return SizeCalc{}, false
	}

	leverage = math.Round(leverage)
	if leverage < 1 {
		leverage = 1
	}

	return SizeCalc{
		Notional:  notional,
		Leverage:  leverage,
		Quantity:  qty,
		RiskUSD:   riskUSD,
		SLDistPct: slDistPct,
	}, true
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
