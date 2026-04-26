package main

import (
	"context"
	"fmt"
	"time"

	"j_ai_trade/brokers/binance"
	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/marketdata"
	"j_ai_trade/trading/models"
)

// historicalFetcher pulls multi-TF candles ending at a simulated wall-
// clock time. Unlike the live BinanceFetcher (which always asks for
// "the most recent N"), this one passes endTime so the model sees only
// data that existed at the moment we're replaying.
//
// Forward-walk M1 candles for outcome resolution use the same path
// (FetchM1Forward) — Binance's startTime+limit gives us up to 1500 M1
// bars per call, plenty for a 4-hour outcome window (240 bars).
type historicalFetcher struct {
	bs *binance.BinanceService
}

func newHistoricalFetcher(bs *binance.BinanceService) *historicalFetcher {
	return &historicalFetcher{bs: bs}
}

// FetchSnapshotAt mirrors marketdata.BinanceFetcher.Fetch but pins
// endTime to `at`. Result is shaped identical to the live path so we
// can feed it straight into market.Build.
func (f *historicalFetcher) FetchSnapshotAt(ctx context.Context, symbol string, at time.Time, required map[models.Timeframe]int) (models.MarketData, error) {
	out := models.MarketData{Symbol: symbol, Candles: map[models.Timeframe][]baseCandle.BaseCandle{}}
	for tf, minCount := range required {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		// Same +20 cushion the live fetcher uses for indicator warm-up
		// boundary safety.
		limit := minCount + 20
		raw, err := f.bs.FetchCandlesEndingAt(ctx, symbol, tf, at, limit)
		if err != nil {
			return out, fmt.Errorf("historical fetch %s %s: %w", symbol, tf, err)
		}
		out.Candles[tf] = marketdata.ConvertBinanceCandles(symbol, raw)
	}
	return out, nil
}

// FetchM1Forward pulls M1 candles strictly AFTER `from` so the outcome
// walker can see whether TP or SL hit first within a few hours. We ask
// for `windowBars` bars; one Binance call returns ≤1500, comfortably
// over our 240-minute (=4h) ceiling. Return value preserves OpenTime
// ordering ascending.
func (f *historicalFetcher) FetchM1Forward(ctx context.Context, symbol string, from time.Time, windowBars int) ([]baseCandle.BaseCandle, error) {
	// Use endTime = from + window so we get bars in [from, from+window].
	// Binance's startTime is inclusive of the bar's open; passing endTime
	// ≈ from + window·1m gives the right tail. Some bars before `from`
	// may sneak in; we filter them out below.
	end := from.Add(time.Duration(windowBars+5) * time.Minute)
	raw, err := f.bs.FetchCandlesEndingAt(ctx, symbol, models.TF_M1, end, windowBars+10)
	if err != nil {
		return nil, fmt.Errorf("forward M1 fetch: %w", err)
	}
	conv := marketdata.ConvertBinanceCandles(symbol, raw)
	// Trim anything that closed at-or-before `from` — those are part of
	// the snapshot, not the future.
	out := make([]baseCandle.BaseCandle, 0, len(conv))
	for _, c := range conv {
		if c.OpenTime.After(from) {
			out = append(out, c)
		}
	}
	return out, nil
}
