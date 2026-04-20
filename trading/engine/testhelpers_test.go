package engine

import (
	"context"
	"math"
	"math/rand"
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/models"
)

// makeCandles builds N candles from a close-price function. Open/high/low are
// derived from close with ±0.3% synthetic spread — small enough that the close
// drives all indicator behavior.
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

func uptrendCandles(n int, startPx, step float64) []baseCandle.BaseCandle {
	return makeCandles(n, func(i int) float64 { return startPx + float64(i)*step })
}

func downtrendCandles(n int, startPx, step float64) []baseCandle.BaseCandle {
	return makeCandles(n, func(i int) float64 { return startPx - float64(i)*step })
}

func rangeCandles(n int, mid, amp float64) []baseCandle.BaseCandle {
	// Random noise around mid — produces low ADX (no sustained direction) and
	// EMAs that converge to the mean, so the regime reads as RANGE cleanly.
	// Sine waves can instead register as a weak trend at the last bar.
	r := rand.New(rand.NewSource(42))
	return makeCandles(n, func(_ int) float64 {
		return mid + (r.Float64()-0.5)*2*amp
	})
}

// fakeStrategy is a test double that returns a pre-programmed vote. It lets us
// test ensemble/consensus logic in isolation from real indicator math.
type fakeStrategy struct {
	name    string
	regimes []models.Regime
	vote    models.StrategyVote
	err     error
}

func (f *fakeStrategy) Name() string                           { return f.name }
func (f *fakeStrategy) RequiredTimeframes() []models.Timeframe { return []models.Timeframe{models.TF_H1} }
func (f *fakeStrategy) MinCandles() map[models.Timeframe]int {
	return map[models.Timeframe]int{models.TF_H1: 0}
}
func (f *fakeStrategy) UsesFundamental() bool          { return false }
func (f *fakeStrategy) ActiveRegimes() []models.Regime { return f.regimes }
func (f *fakeStrategy) Analyze(_ context.Context, _ StrategyInput) (*models.StrategyVote, error) {
	if f.err != nil {
		return nil, f.err
	}
	v := f.vote
	v.Name = f.name
	return &v, nil
}
