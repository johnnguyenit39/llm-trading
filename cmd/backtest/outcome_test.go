package main

import (
	"testing"
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/modules/advisor/biz"
)

// bar is a tiny constructor for test candles. Time fields are derived
// from the index so order-sensitive walks have realistic CloseTime gaps.
func bar(i int, open, high, low, close float64) baseCandle.BaseCandle {
	t := time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC)
	return baseCandle.BaseCandle{
		OpenTime:  t,
		Open:      open, High: high, Low: low, Close: close,
		CloseTime: t.Add(59 * time.Second),
	}
}

func TestResolveOutcome_BuyHitsTPFirst(t *testing.T) {
	d := &biz.DecisionPayload{Action: "BUY", Entry: 100, StopLoss: 99, TakeProfit: 102}
	forward := []baseCandle.BaseCandle{
		bar(0, 100, 100.5, 99.8, 100.2),
		bar(1, 100.2, 102.1, 100.0, 101.9), // hits TP=102 via wick
	}
	got := resolveOutcome(d, forward)
	if got.Kind != outcomeTP {
		t.Fatalf("want tp, got %s", got.Kind)
	}
	if got.BarsToOutcome != 2 {
		t.Fatalf("want bars=2, got %d", got.BarsToOutcome)
	}
}

func TestResolveOutcome_SellHitsSLFirst(t *testing.T) {
	d := &biz.DecisionPayload{Action: "SELL", Entry: 100, StopLoss: 101, TakeProfit: 98}
	forward := []baseCandle.BaseCandle{
		bar(0, 100, 101.2, 99.7, 100.5), // wick to 101.2 ≥ SL
	}
	got := resolveOutcome(d, forward)
	if got.Kind != outcomeSL {
		t.Fatalf("want sl, got %s", got.Kind)
	}
}

func TestResolveOutcome_TimeoutWhenNeitherHit(t *testing.T) {
	d := &biz.DecisionPayload{Action: "BUY", Entry: 100, StopLoss: 95, TakeProfit: 110}
	forward := []baseCandle.BaseCandle{
		bar(0, 100, 100.5, 99.8, 100.2),
		bar(1, 100.2, 100.7, 100.0, 100.5),
		bar(2, 100.5, 100.9, 100.3, 100.6),
	}
	got := resolveOutcome(d, forward)
	if got.Kind != outcomeTimeout {
		t.Fatalf("want timeout, got %s", got.Kind)
	}
	// MFE ≈ 100.9 - 100 = 0.9
	if got.MFE < 0.85 || got.MFE > 0.95 {
		t.Fatalf("unexpected MFE: %.4f", got.MFE)
	}
}

func TestResolveOutcome_SameBarBothSidesAssumesSL(t *testing.T) {
	// Wick covers BOTH SL and TP in a single bar — conservative
	// convention treats this as SL hit. (M1 granularity can't tell
	// intra-bar order.)
	d := &biz.DecisionPayload{Action: "BUY", Entry: 100, StopLoss: 99, TakeProfit: 101}
	forward := []baseCandle.BaseCandle{
		bar(0, 100, 101.5, 98.5, 100), // both stops touched
	}
	got := resolveOutcome(d, forward)
	if got.Kind != outcomeSL {
		t.Fatalf("want sl on tie, got %s", got.Kind)
	}
}

func TestResolveOutcome_NilDecisionIsError(t *testing.T) {
	got := resolveOutcome(nil, []baseCandle.BaseCandle{bar(0, 1, 2, 0, 1)})
	if got.Kind != outcomeError {
		t.Fatalf("want error for nil decision, got %s", got.Kind)
	}
}

func TestResolveOutcome_EmptyForwardIsError(t *testing.T) {
	d := &biz.DecisionPayload{Action: "BUY", Entry: 100, StopLoss: 99, TakeProfit: 101}
	got := resolveOutcome(d, nil)
	if got.Kind != outcomeError {
		t.Fatalf("want error for empty forward, got %s", got.Kind)
	}
}
