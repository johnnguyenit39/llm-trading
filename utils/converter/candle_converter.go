package model

import (
	binanceRepository "j_ai_trade/brokers/binance/repository"
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
