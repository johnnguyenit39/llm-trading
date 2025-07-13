package model

import (
	binanceRepository "j_ai_trade/brokers/binance/repository"
	twelveRepository "j_ai_trade/brokers/twelve/repository"
	baseCandleModel "j_ai_trade/common"
)

// ConvertBinanceCandleToBase converts a Binance candle to a base candle
func ConvertBinanceCandleToBase(binanceCandle binanceRepository.BinanceCandle) baseCandleModel.BaseCandle {
	return baseCandleModel.BaseCandle{
		Symbol:    binanceCandle.Symbol,
		OpenTime:  binanceCandle.OpenTime,
		Open:      binanceCandle.Open,
		High:      binanceCandle.High,
		Low:       binanceCandle.Low,
		Close:     binanceCandle.Close,
		Volume:    binanceCandle.Volume,
		CloseTime: binanceCandle.CloseTime,
	}
}

// ConvertBinanceCandlesToBase converts a slice of Binance candles to base candles
func ConvertBinanceCandlesToBase(binanceCandles []binanceRepository.BinanceCandle) []baseCandleModel.BaseCandle {
	baseCandles := make([]baseCandleModel.BaseCandle, len(binanceCandles))
	for i, candle := range binanceCandles {
		baseCandles[i] = ConvertBinanceCandleToBase(candle)
	}
	return baseCandles
}

// ConvertTwelveCandleToBase converts a Twelve candle to a base candle
func ConvertTwelveCandleToBase(twelveCandle twelveRepository.TwelveCandle) baseCandleModel.BaseCandle {
	return baseCandleModel.BaseCandle{
		OpenTime:  twelveCandle.DateTime,
		Open:      twelveCandle.Open,
		High:      twelveCandle.High,
		Low:       twelveCandle.Low,
		Close:     twelveCandle.Close,
		Volume:    0,                     // Twelve Data doesn't provide volume
		CloseTime: twelveCandle.DateTime, // Using DateTime as CloseTime since Twelve doesn't provide separate close time
	}
}

// ConvertTwelveCandlesToBase converts a slice of Twelve candles to base candles
func ConvertTwelveCandlesToBase(twelveCandles []twelveRepository.TwelveCandle) []baseCandleModel.BaseCandle {
	baseCandles := make([]baseCandleModel.BaseCandle, len(twelveCandles))
	for i, candle := range twelveCandles {
		baseCandles[i] = ConvertTwelveCandleToBase(candle)
	}
	return baseCandles
}
