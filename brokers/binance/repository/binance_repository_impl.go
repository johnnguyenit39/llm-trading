package repository

import (
	"context"
	"encoding/json"
	"fmt"
	utils "j_ai_trade/brokers/binance/utils"
	"net/http"
	"strconv"
	"time"
)

const (
	// defaultBinanceURL points at Binance USDT-Margined Futures, where
	// our klines come from (best gold liquidity / 24h coverage).
	defaultBinanceURL = "https://fapi.binance.com"

	// defaultBinanceSpotURL is Binance Spot — used only for the small
	// /api/v3/ticker/price calls we need to convert USDT-denominated
	// prices into USD (the USDTUSD pair lives on spot, not futures).
	defaultBinanceSpotURL = "https://api.binance.com"
)

type binanceRepositoryImpl struct {
	baseURL string // futures (klines)
	spotURL string // spot (ticker)
	client  *http.Client
}

// NewBinanceRepository creates a new BinanceRepository instance with default Binance API URL
func NewBinanceRepository() BinanceRepository {
	return &binanceRepositoryImpl{
		baseURL: defaultBinanceURL,
		spotURL: defaultBinanceSpotURL,
		client:  &http.Client{},
	}
}

func (r *binanceRepositoryImpl) FetchCandles(ctx context.Context, symbol string, interval string, limit int) ([]BinanceCandle, error) {

	formatedSymbol := utils.ConvertPair(symbol)

	url := fmt.Sprintf("%s/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", r.baseURL, formatedSymbol, interval, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rawCandles [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawCandles); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	candles := make([]BinanceCandle, 0, len(rawCandles))
	for _, raw := range rawCandles {
		openTime := int64(raw[0].(float64))
		closeTime := int64(raw[6].(float64))

		open, _ := strconv.ParseFloat(raw[1].(string), 64)
		high, _ := strconv.ParseFloat(raw[2].(string), 64)
		low, _ := strconv.ParseFloat(raw[3].(string), 64)
		close, _ := strconv.ParseFloat(raw[4].(string), 64)
		volume, _ := strconv.ParseFloat(raw[5].(string), 64)

		candles = append(candles, BinanceCandle{
			Symbol:    symbol,
			OpenTime:  time.Unix(0, openTime*int64(time.Millisecond)),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: time.Unix(0, closeTime*int64(time.Millisecond)),
		})
	}

	return candles, nil
}

// FetchSpotTickerPrice hits Binance spot's /api/v3/ticker/price endpoint
// for a single symbol and returns the latest trade price. Validates the
// HTTP status and that the parsed price is positive — a zero or
// negative price would propagate as a silent multiply-by-zero into
// every converted candle, which is much worse than a loud error.
func (r *binanceRepositoryImpl) FetchSpotTickerPrice(ctx context.Context, symbol string) (float64, error) {
	formattedSymbol := utils.ConvertPair(symbol)
	url := fmt.Sprintf("%s/api/v3/ticker/price?symbol=%s", r.spotURL, formattedSymbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	price, err := strconv.ParseFloat(payload.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price %q: %w", payload.Price, err)
	}
	if price <= 0 {
		return 0, fmt.Errorf("non-positive ticker price %v for %s", price, symbol)
	}
	return price, nil
}
