package models

import baseCandleModel "j_ai_trade/common"

// MarketData carries multi-timeframe candles for a single symbol.
type MarketData struct {
	Symbol  string
	Candles map[Timeframe][]baseCandleModel.BaseCandle
}

// Get returns the candle slice for a given timeframe. Returns nil if not present.
func (m MarketData) Get(tf Timeframe) []baseCandleModel.BaseCandle {
	if m.Candles == nil {
		return nil
	}
	return m.Candles[tf]
}

// Has returns true if the timeframe has enough candles.
func (m MarketData) Has(tf Timeframe, minCount int) bool {
	return len(m.Get(tf)) >= minCount
}
