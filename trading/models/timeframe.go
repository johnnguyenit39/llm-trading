package models

type Timeframe string

const (
	TF_M1  Timeframe = "M1"
	TF_M5  Timeframe = "M5"
	TF_M15 Timeframe = "M15"
	TF_H1  Timeframe = "H1"
	TF_H4  Timeframe = "H4"
	TF_D1  Timeframe = "D1"
	TF_W1  Timeframe = "W1"
)

// BinanceInterval returns the Binance API interval string for a timeframe.
func (tf Timeframe) BinanceInterval() string {
	switch tf {
	case TF_M1:
		return "1m"
	case TF_M5:
		return "5m"
	case TF_M15:
		return "15m"
	case TF_H1:
		return "1h"
	case TF_H4:
		return "4h"
	case TF_D1:
		return "1d"
	case TF_W1:
		return "1w"
	}
	return ""
}
