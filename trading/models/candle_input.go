package models

import baseCandleModel "j_ai_trade/common"

type CandleInput struct {
	M15Candles []baseCandleModel.BaseCandle // M15 candles for EMA 200 trend filter
	M5Candles  []baseCandleModel.BaseCandle // M5 candles for EMA 50 and M5-M1 alignment
	M1Candles  []baseCandleModel.BaseCandle // M1 candles for RSI and patterns (matching TradingView)
	H1Candles  []baseCandleModel.BaseCandle // H1 candles for trend analysis
	H4Candles  []baseCandleModel.BaseCandle // H4 candles for major trend
	D1Candles  []baseCandleModel.BaseCandle // D1 candles for daily trend
}
