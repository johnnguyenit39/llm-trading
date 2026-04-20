package strategies

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// ---- helpers ----

func makeCandles(n int, closeFn func(i int) float64) []baseCandle.BaseCandle {
	out := make([]baseCandle.BaseCandle, n)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		c := closeFn(i)
		o := c
		if i > 0 {
			o = out[i-1].Close
		}
		hi := math.Max(o, c) * 1.003
		lo := math.Min(o, c) * 0.997
		out[i] = baseCandle.BaseCandle{
			Symbol:    "TESTUSDT",
			OpenTime:  start.Add(time.Duration(i) * time.Hour),
			Open:      o,
			High:      hi,
			Low:       lo,
			Close:     c,
			Volume:    100,
			CloseTime: start.Add(time.Duration(i+1)*time.Hour - time.Second),
		}
	}
	return out
}

func uptrend(n int, startPx, step float64) []baseCandle.BaseCandle {
	return makeCandles(n, func(i int) float64 { return startPx + float64(i)*step })
}

func rangeBars(n int, mid, amp float64) []baseCandle.BaseCandle {
	// Random noise → low ADX and balanced EMAs; cleaner "range" signal than a
	// deterministic sine which can register as a tiny trend at the last bar.
	r := rand.New(rand.NewSource(42))
	return makeCandles(n, func(_ int) float64 {
		return mid + (r.Float64()-0.5)*2*amp
	})
}

func market(entryTF models.Timeframe, entry []baseCandle.BaseCandle, htfTF models.Timeframe, htf []baseCandle.BaseCandle) models.MarketData {
	m := models.MarketData{
		Symbol:  "BTCUSDT",
		Candles: map[models.Timeframe][]baseCandle.BaseCandle{entryTF: entry},
	}
	if htf != nil {
		m.Candles[htfTF] = htf
	}
	return m
}

// ---- TrendFollow ----

func TestTrendFollow_FiresOnUptrendTaggingEMA20(t *testing.T) {
	n := 250
	entry := uptrend(n, 100, 0.5)
	trend := uptrend(n, 100, 0.5)

	// Strategy reads entry[:n-1] after ClosedCandles, so its "last" is bar n-2.
	// Patch that bar to sit within [EMA20, EMA20+0.5*ATR] — computed from the
	// preceding series so we know the target value.
	base := entry[:n-2]
	closes := indicators.Closes(base)
	ema20 := indicators.EMA(closes, 20)
	atr := indicators.ATR(base, 14)
	target := ema20 + 0.3*atr

	entry[n-2].Close = target
	entry[n-2].Open = target
	entry[n-2].High = target + 0.1*atr
	entry[n-2].Low = target - 0.1*atr

	s := NewTrendFollow(models.TF_H1, models.TF_H4)
	vote, err := s.Analyze(context.Background(), engine.StrategyInput{
		Market:       market(models.TF_H1, entry, models.TF_H4, trend),
		Equity:       1000,
		CurrentPrice: target,
		EntryTF:      models.TF_H1,
		Regime:       models.RegimeTrendUp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if vote.Direction != models.DirectionBuy {
		t.Fatalf("expected BUY, got %s (reason=%s)", vote.Direction, vote.Reason)
	}
	if !(vote.StopLoss < vote.Entry && vote.Entry < vote.TakeProfit) {
		t.Errorf("incoherent BUY: entry=%v SL=%v TP=%v", vote.Entry, vote.StopLoss, vote.TakeProfit)
	}
	if vote.Confidence < 60 || vote.Confidence > 90 {
		t.Errorf("confidence out of 60..90: %v", vote.Confidence)
	}
}

func TestTrendFollow_NoFireOnRange(t *testing.T) {
	entry := rangeBars(250, 100, 2)
	trend := rangeBars(250, 100, 2)
	s := NewTrendFollow(models.TF_H1, models.TF_H4)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, entry, models.TF_H4, trend),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeRange,
	})
	if vote.Direction != models.DirectionNone {
		t.Errorf("expected NONE on range, got %s (reason=%s)", vote.Direction, vote.Reason)
	}
}

func TestTrendFollow_NoFireInsufficientCandles(t *testing.T) {
	entry := uptrend(100, 100, 0.5) // < 210 required
	trend := uptrend(100, 100, 0.5)
	s := NewTrendFollow(models.TF_H1, models.TF_H4)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, entry, models.TF_H4, trend),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeTrendUp,
	})
	if vote.Direction != models.DirectionNone {
		t.Errorf("expected NONE on insufficient data, got %s", vote.Direction)
	}
}

// ---- MeanReversion ----

func TestMeanReversion_NoFireInStrongTrend(t *testing.T) {
	// Strong uptrend → ADX high → mean-rev refuses.
	entry := uptrend(80, 100, 1.0)
	s := NewMeanReversion(models.TF_H1)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, entry, "", nil),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeTrendUp,
	})
	if vote.Direction != models.DirectionNone {
		t.Errorf("expected NONE in strong trend, got %s (reason=%s)", vote.Direction, vote.Reason)
	}
}

func TestMeanReversion_NoFireAtMidOfRange(t *testing.T) {
	// Range market but current price sitting near the Bollinger middle — no
	// extreme → abstain.
	entry := rangeBars(80, 100, 0.5)
	s := NewMeanReversion(models.TF_H1)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, entry, "", nil),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeRange,
	})
	if vote.Direction != models.DirectionNone {
		t.Errorf("expected NONE at mid-range, got %s (reason=%s)", vote.Direction, vote.Reason)
	}
}

func TestMeanReversion_CoherentSLTPIfFires(t *testing.T) {
	// Engineered extreme: force last closed bar way below lower BB. Even if
	// RSI gate or ADX gate rejects under smoothing, any *fired* vote must
	// satisfy SL < Entry < TP. This is an invariant test — accepts NONE,
	// rejects incoherent fires.
	n := 80
	entry := rangeBars(n, 100, 0.3)
	base := entry[:n-2]
	closes := indicators.Closes(base)
	_, _, lower := indicators.BollingerBands(closes, 20, 2.0)
	target := lower - 2.0 // well below
	entry[n-2].Close = target
	entry[n-2].Open = entry[n-3].Close
	entry[n-2].High = entry[n-3].Close
	entry[n-2].Low = target - 0.5

	s := NewMeanReversion(models.TF_H1)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, entry, "", nil),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeRange,
	})
	if vote.Direction == models.DirectionBuy {
		if !(vote.StopLoss < vote.Entry && vote.Entry < vote.TakeProfit) {
			t.Errorf("incoherent BUY: entry=%v SL=%v TP=%v", vote.Entry, vote.StopLoss, vote.TakeProfit)
		}
	}
}

// ---- Breakout ----

func TestBreakout_NoFireWithoutVolumeExpansion(t *testing.T) {
	// Price breaks Donchian but volume is flat → must abstain.
	n := 60
	entry := rangeBars(n, 100, 1) // tight range
	// Force last closed bar to break above prior 20-bar high with FLAT volume.
	priorHigh := entry[0].High
	for _, c := range entry[:n-2] {
		if c.High > priorHigh {
			priorHigh = c.High
		}
	}
	entry[n-2].Close = priorHigh * 1.03
	entry[n-2].High = entry[n-2].Close
	entry[n-2].Open = priorHigh
	entry[n-2].Volume = 100 // same as average

	s := NewBreakout(models.TF_H1, 20)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, entry, "", nil),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeTrendUp,
	})
	if vote.Direction != models.DirectionNone {
		t.Errorf("expected NONE without volume expansion, got %s", vote.Direction)
	}
}

func TestBreakout_FiresOnBreakoutWithVolumeAndATRExpansion(t *testing.T) {
	// Tight compression for the first ~40 bars (low ATR), last ~15 bars
	// wider ranges (ATR expansion), last closed bar breaks above prior high
	// with a 3x volume spike.
	n := 60
	entry := makeCandles(n, func(i int) float64 {
		if i < n-15 {
			return 100 + math.Sin(float64(i)*0.5)*0.3 // very tight
		}
		// Recent ~15 bars: slightly wider drift up
		return 100 + float64(i-(n-15))*0.2
	})
	// Widen recent bars' HL to inflate recent ATR.
	for k := 15; k >= 2; k-- {
		entry[n-k].High = entry[n-k].Close + 0.5
		entry[n-k].Low = entry[n-k].Close - 0.5
	}
	// Engineer the last closed bar (n-2) as a confirmed breakout.
	priorHigh := 0.0
	for _, c := range entry[:n-2] {
		if c.High > priorHigh {
			priorHigh = c.High
		}
	}
	breakPx := priorHigh + 1.5
	entry[n-2].Open = priorHigh
	entry[n-2].Close = breakPx
	entry[n-2].High = breakPx + 0.1
	entry[n-2].Low = priorHigh - 0.2
	entry[n-2].Volume = 400 // 4x average

	s := NewBreakout(models.TF_H1, 20)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, entry, "", nil),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeTrendUp,
	})
	if vote.Direction != models.DirectionBuy {
		t.Fatalf("expected BUY on engineered breakout, got %s (reason=%s)", vote.Direction, vote.Reason)
	}
	if !(vote.StopLoss < vote.Entry && vote.Entry < vote.TakeProfit) {
		t.Errorf("incoherent BUY: entry=%v SL=%v TP=%v", vote.Entry, vote.StopLoss, vote.TakeProfit)
	}
}

// ---- Structure ----

func TestStructure_NoFireInPlainRange(t *testing.T) {
	// Pure range without a clean break of structure.
	bars := rangeBars(40, 100, 2)
	s := NewStructure(models.TF_H1, models.TF_H4, 3)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, bars, models.TF_H4, bars),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeRange,
	})
	if vote.Direction != models.DirectionNone {
		t.Errorf("expected NONE without BOS, got %s (reason=%s)", vote.Direction, vote.Reason)
	}
}

func TestStructure_NoFireInsufficientCandles(t *testing.T) {
	bars := uptrend(10, 100, 0.5)
	s := NewStructure(models.TF_H1, models.TF_H4, 3)
	vote, _ := s.Analyze(context.Background(), engine.StrategyInput{
		Market:  market(models.TF_H1, bars, models.TF_H4, bars),
		Equity:  1000,
		EntryTF: models.TF_H1,
		Regime:  models.RegimeTrendUp,
	})
	if vote.Direction != models.DirectionNone {
		t.Errorf("expected NONE on insufficient data, got %s", vote.Direction)
	}
}
