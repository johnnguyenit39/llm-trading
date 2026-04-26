// Package marketdata is the shared candle-fetch layer used by both the
// cron signal broadcaster and the advisor chat bot. Keeping this outside
// modules/advisor/ and cron_jobs/ prevents duplication and makes it easy
// to swap Binance for another exchange later (add a new implementation of
// CandleFetcher; callers don't change).
package marketdata

import (
	"context"
	"fmt"
	"strings"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/models"
)

// CandleFetcher abstracts the exchange REST client. The advisor and cron
// depend on this interface so tests can feed canned candles and the
// concrete Binance impl lives in exactly one place.
type CandleFetcher interface {
	// Fetch returns candles for each requested timeframe, keyed by the
	// Timeframe value. The returned MarketData carries the same Symbol
	// string that was passed in. Errors are returned for transport or
	// API failures; an empty timeframe map means none of the requested
	// TFs produced candles.
	Fetch(ctx context.Context, symbol string, required map[models.Timeframe]int) (models.MarketData, error)
}

// BinanceFetcher is the production CandleFetcher backed by Binance
// public REST (no API key required for klines).
type BinanceFetcher struct {
	bs *binance.BinanceService
}

// NewBinanceFetcher wraps an existing BinanceService. It is safe to share
// a single service across goroutines.
func NewBinanceFetcher(bs *binance.BinanceService) *BinanceFetcher {
	return &BinanceFetcher{bs: bs}
}

// Fetch pulls the requested candle counts per timeframe. We fetch
// `minCount + 20` so indicators with warm-up periods (ADX-28, EMA-200)
// have a cushion against off-by-one boundaries.
//
// Forex-flavoured conversion: when the symbol is USDT-quoted (e.g.
// XAUUSDT) we additionally fetch the live USDTUSD spot rate and scale
// every OHLC value by it, so callers see prices in real US dollars
// rather than Tether. Volume stays as-is — it's denominated in the
// base asset (oz of gold), not the quote. The output Symbol is
// renamed accordingly (XAUUSDT → XAUUSD) so the rest of the pipeline
// labels prices honestly. If the rate fetch fails we surface the
// error rather than silently returning USDT-denominated candles —
// for a forex trader, a 1-2 pip blind spot is worse than a fallback
// to chat-only.
func (f *BinanceFetcher) Fetch(ctx context.Context, symbol string, required map[models.Timeframe]int) (models.MarketData, error) {
	out := models.MarketData{Symbol: symbol, Candles: map[models.Timeframe][]baseCandle.BaseCandle{}}
	for tf, minCount := range required {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		limit := minCount + 20
		var (
			candles []repository.BinanceCandle
			err     error
		)
		switch tf {
		case models.TF_M1:
			candles, err = f.bs.Fetch1mCandles(ctx, symbol, limit)
		case models.TF_M5:
			candles, err = f.bs.Fetch5mCandles(ctx, symbol, limit)
		case models.TF_M15:
			candles, err = f.bs.Fetch15mCandles(ctx, symbol, limit)
		case models.TF_H1:
			candles, err = f.bs.Fetch1hCandles(ctx, symbol, limit)
		case models.TF_H4:
			candles, err = f.bs.Fetch4hCandles(ctx, symbol, limit)
		case models.TF_D1:
			candles, err = f.bs.Fetch1dCandles(ctx, symbol, limit)
		default:
			continue
		}
		if err != nil {
			return out, fmt.Errorf("fetch %s %s: %w", symbol, tf, err)
		}
		out.Candles[tf] = ConvertBinanceCandles(symbol, candles)
	}
	if strings.HasSuffix(symbol, "USDT") {
		rate, err := f.bs.FetchSpotTickerPrice(ctx, "USDTUSD")
		if err != nil {
			return out, fmt.Errorf("fetch USDTUSD rate: %w", err)
		}
		usdSymbol := strings.TrimSuffix(symbol, "T") // XAUUSDT -> XAUUSD
		ApplyUSDTtoUSDRate(&out, rate, usdSymbol)
	}
	return out, nil
}

// ApplyUSDTtoUSDRate scales every candle's OHLC by the given USDTUSD
// rate, in place, and renames the MarketData's Symbol (and each
// candle's Symbol) to the USD-denominated form. Volume is left alone —
// it counts base-asset units, not quote currency.
//
// Exported so unit tests and alt-broker adapters can reuse it without
// re-deriving the multiplier semantics.
func ApplyUSDTtoUSDRate(md *models.MarketData, rate float64, usdSymbol string) {
	if md == nil || rate <= 0 {
		return
	}
	for tf, candles := range md.Candles {
		for i := range candles {
			candles[i].Open *= rate
			candles[i].High *= rate
			candles[i].Low *= rate
			candles[i].Close *= rate
			candles[i].Symbol = usdSymbol
		}
		md.Candles[tf] = candles
	}
	md.Symbol = usdSymbol
}

// ConvertBinanceCandles normalises exchange-specific candles into the
// engine's BaseCandle shape. Exported so tests and alt-exchange adapters
// can reuse the mapping without re-deriving it.
func ConvertBinanceCandles(symbol string, src []repository.BinanceCandle) []baseCandle.BaseCandle {
	out := make([]baseCandle.BaseCandle, len(src))
	for i, c := range src {
		out[i] = baseCandle.BaseCandle{
			Symbol:    symbol,
			OpenTime:  c.OpenTime,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			CloseTime: c.CloseTime,
		}
	}
	return out
}
