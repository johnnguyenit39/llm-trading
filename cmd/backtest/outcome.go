package main

import (
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/modules/advisor/biz"
)

// outcome is what walking the future M1 candles reveals about a
// decision. We keep the enum tight so aggregation is unambiguous: the
// trade either reached TP first, SL first, or neither within the
// window. Anything else (parse failed, no candles) is reported as
// outcomeError so it never silently inflates "win" or "loss" counts.
type outcomeKind string

const (
	outcomeTP      outcomeKind = "tp"      // hit take-profit before stop-loss
	outcomeSL      outcomeKind = "sl"      // hit stop-loss before take-profit
	outcomeTimeout outcomeKind = "timeout" // window expired without hitting either
	outcomeError   outcomeKind = "error"   // walker couldn't run (no candles, etc.)
)

// outcomeResult is the per-decision result of the forward walk. We
// capture MFE/MAE in price units so the report can flag SLs that were
// unnecessarily wide (low MAE, premature SL hits) and TPs that were
// reachable but the trade closed too early (would need exit logic to
// matter — backtest just records).
type outcomeResult struct {
	Kind          outcomeKind   `json:"kind"`
	BarsToOutcome int           `json:"bars_to_outcome"` // M1 bars; 0 if Kind != tp/sl
	TimeToOutcome time.Duration `json:"time_to_outcome"`
	MFE           float64       `json:"mfe"` // max favourable excursion (price-units)
	MAE           float64       `json:"mae"` // max adverse excursion (price-units)
}

// resolveOutcome walks M1 candles `bars` forward from `from` and reports
// what would have happened to a trade with the given action/entry/SL/TP.
// Crossing logic is "high/low touch" — i.e. a wick that pierces the
// level counts as a hit. This matches retail broker fills (broker stops
// trigger on wicks) and is the standard backtest convention.
//
// When entry isn't yet reached when the window opens, we still walk:
// the digest's "entry" is a limit price that the LLM picked from
// just-closed market structure, so the very next M1 candle usually
// crosses it. We do NOT require an explicit entry-fill bar — outcome
// computation starts from the first forward bar.
func resolveOutcome(d *biz.DecisionPayload, forward []baseCandle.BaseCandle) outcomeResult {
	if d == nil || len(forward) == 0 {
		return outcomeResult{Kind: outcomeError}
	}
	switch d.Action {
	case "BUY":
		return walkBuy(d, forward)
	case "SELL":
		return walkSell(d, forward)
	default:
		return outcomeResult{Kind: outcomeError}
	}
}

func walkBuy(d *biz.DecisionPayload, forward []baseCandle.BaseCandle) outcomeResult {
	mfe, mae := 0.0, 0.0
	for i, c := range forward {
		if hi := c.High - d.Entry; hi > mfe {
			mfe = hi
		}
		if lo := d.Entry - c.Low; lo > mae {
			mae = lo
		}
		// Same-bar tie-break: if the bar's range covers BOTH SL and TP
		// we conservatively assume SL hit first. Real fills depend on
		// intra-bar order, which we don't have at M1 granularity. This
		// matches industry-standard backtest convention for stop orders.
		hitSL := c.Low <= d.StopLoss
		hitTP := c.High >= d.TakeProfit
		switch {
		case hitSL && hitTP:
			return outcomeResult{
				Kind:          outcomeSL,
				BarsToOutcome: i + 1,
				TimeToOutcome: c.CloseTime.Sub(forward[0].OpenTime),
				MFE:           mfe,
				MAE:           mae,
			}
		case hitSL:
			return outcomeResult{
				Kind:          outcomeSL,
				BarsToOutcome: i + 1,
				TimeToOutcome: c.CloseTime.Sub(forward[0].OpenTime),
				MFE:           mfe,
				MAE:           mae,
			}
		case hitTP:
			return outcomeResult{
				Kind:          outcomeTP,
				BarsToOutcome: i + 1,
				TimeToOutcome: c.CloseTime.Sub(forward[0].OpenTime),
				MFE:           mfe,
				MAE:           mae,
			}
		}
	}
	return outcomeResult{Kind: outcomeTimeout, MFE: mfe, MAE: mae}
}

func walkSell(d *biz.DecisionPayload, forward []baseCandle.BaseCandle) outcomeResult {
	mfe, mae := 0.0, 0.0
	for i, c := range forward {
		if lo := d.Entry - c.Low; lo > mfe {
			mfe = lo
		}
		if hi := c.High - d.Entry; hi > mae {
			mae = hi
		}
		hitSL := c.High >= d.StopLoss
		hitTP := c.Low <= d.TakeProfit
		switch {
		case hitSL && hitTP:
			return outcomeResult{
				Kind:          outcomeSL,
				BarsToOutcome: i + 1,
				TimeToOutcome: c.CloseTime.Sub(forward[0].OpenTime),
				MFE:           mfe,
				MAE:           mae,
			}
		case hitSL:
			return outcomeResult{
				Kind:          outcomeSL,
				BarsToOutcome: i + 1,
				TimeToOutcome: c.CloseTime.Sub(forward[0].OpenTime),
				MFE:           mfe,
				MAE:           mae,
			}
		case hitTP:
			return outcomeResult{
				Kind:          outcomeTP,
				BarsToOutcome: i + 1,
				TimeToOutcome: c.CloseTime.Sub(forward[0].OpenTime),
				MFE:           mfe,
				MAE:           mae,
			}
		}
	}
	return outcomeResult{Kind: outcomeTimeout, MFE: mfe, MAE: mae}
}
