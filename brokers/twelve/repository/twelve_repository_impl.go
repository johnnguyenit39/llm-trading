package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

type twelveResponse struct {
	Meta struct {
		Symbol       string `json:"symbol"`
		Interval     string `json:"interval"`
		CurrencyBase string `json:"currency_base"`
		Type         string `json:"type"`
	} `json:"meta"`
	Values []struct {
		DateTime string `json:"datetime"`
		Open     string `json:"open"`
		High     string `json:"high"`
		Low      string `json:"low"`
		Close    string `json:"close"`
	} `json:"values"`
	Status string `json:"status"`
}

type TwelveRepositoryImpl struct {
	client *http.Client
	apiKey string
}

func NewTwelveRepositoryImpl() *TwelveRepositoryImpl {
	return &TwelveRepositoryImpl{
		client: &http.Client{},
		apiKey: os.Getenv("TWELVEDATA_KEY"),
	}
}

func (r *TwelveRepositoryImpl) FetchCandles(ctx context.Context, symbol string, interval string, limit int) ([]TwelveCandle, error) {
	url := fmt.Sprintf("https://api.twelvedata.com/time_series?symbol=%s&interval=%s&outputsize=%d&apikey=%s",
		symbol, interval, limit, r.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var data twelveResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if data.Status != "ok" {
		return nil, fmt.Errorf("API returned non-ok status: %s", data.Status)
	}

	candles := make([]TwelveCandle, len(data.Values))
	for i, v := range data.Values {
		datetime, err := time.Parse("2006-01-02 15:04:05", v.DateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse datetime: %w", err)
		}

		open, _ := strconv.ParseFloat(v.Open, 64)
		high, _ := strconv.ParseFloat(v.High, 64)
		low, _ := strconv.ParseFloat(v.Low, 64)
		close, _ := strconv.ParseFloat(v.Close, 64)

		candles[i] = TwelveCandle{
			DateTime: datetime,
			Open:     open,
			High:     high,
			Low:      low,
			Close:    close,
		}
	}

	return candles, nil
}
