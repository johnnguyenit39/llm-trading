package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// RSIScalpingStrategy is designed for ranging and reversal markets
type RSIScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewRSIScalpingStrategy() *RSIScalpingStrategy {
	return &RSIScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "RSI Scalping",
		},
	}
}

func (s *RSIScalpingStrategy) GetDescription() string {
	return "Scalping strategy using RSI for ranging markets. Best for mean reversion and low volatility conditions."
}

func (s *RSIScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketRanging:
		return true
	case common.MarketLowVolatility:
		return true
	default:
		return false
	}
}

func (s *RSIScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 14 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
		volumes[i] = c.Volume
	}

	// Calculate RSI
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}

	// Calculate Stochastic for additional confirmation
	slowK, slowD := talib.Stoch(highs, lows, closes, 14, 3, talib.SMA, 3, talib.SMA)

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate EMA for trend
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)

	// Get latest values
	latestRSI := rsi[len(rsi)-1]
	prevRSI := rsi[len(rsi)-2]
	latestSlowK := slowK[len(slowK)-1]
	latestSlowD := slowD[len(slowD)-1]
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]

	// Calculate market metrics
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	priceVsEMA20 := ((latestPrice - latestEMA20) / latestEMA20) * 100
	volatilityRatio := atrValue / latestPrice * 100

	// Calculate leverage based on volatility and RSI
	leverage := 3.0 // Default for RSI trading
	if volatilityRatio > 3.0 {
		leverage = 2.0
	} else if volatilityRatio > 2.0 {
		leverage = 2.5
	} else if volatilityRatio > 1.0 {
		leverage = 3.0
	} else {
		leverage = 4.0
	}

	// Adjust leverage based on RSI extremes
	if latestRSI < 20 || latestRSI > 80 {
		leverage *= 0.8 // Reduce leverage at extreme RSI levels
	}

	// Cap maximum leverage
	if leverage > 5.0 {
		leverage = 5.0
	}

	// Calculate stop loss and take profit based on volatility
	stopLossDistance := math.Max(atrValue*1.2, latestPrice*0.01) // Minimum 1% stop loss
	takeProfitDistance := math.Max(atrValue*2.0, stopLossDistance*1.5)

	// Calculate position size based on risk
	accountSize := 1000.0
	accountRisk := 0.02
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (stopLossDistance / latestPrice)
	rewardAmount := riskAmount * (takeProfitDistance / stopLossDistance)

	// Trading logic with improved RSI analysis
	if latestRSI < 30 && // Oversold
		prevRSI >= 30 && // RSI crossing up from oversold
		latestSlowK < 20 && // Stochastic oversold
		latestSlowK > latestSlowD && // Stochastic bullish crossover
		volumeStrength > 120 && // Above average volume
		priceVsEMA20 > -1.0 && // Price near or above EMA20
		latestEMA20 > latestEMA50 { // Uptrend confirmation

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 RSI Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• Previous RSI: %.2f\n"+
				"• Stochastic K: %.2f\n"+
				"• Stochastic D: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• RSI oversold bounce setup\n"+
				"• Stochastic confirmation\n"+
				"• Volume and trend support\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Reward Amount: $%.2f\n"+
				"• Position Value: $%.2f",
				latestPrice,
				latestPrice-stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice+takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				takeProfitDistance/stopLossDistance,
				leverage,
				positionSize*100/accountSize,
				latestRSI,
				prevRSI,
				latestSlowK,
				latestSlowD,
				latestEMA20,
				latestEMA50,
				volumeStrength,
				priceVsEMA20,
				volatilityRatio,
				atrValue,
				accountSize,
				riskAmount,
				rewardAmount,
				positionSize,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestRSI > 70 && // Overbought
		prevRSI <= 70 && // RSI crossing down from overbought
		latestSlowK > 80 && // Stochastic overbought
		latestSlowK < latestSlowD && // Stochastic bearish crossover
		volumeStrength > 120 && // Above average volume
		priceVsEMA20 < 1.0 && // Price near or below EMA20
		latestEMA20 < latestEMA50 { // Downtrend confirmation

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 RSI Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• Previous RSI: %.2f\n"+
				"• Stochastic K: %.2f\n"+
				"• Stochastic D: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• RSI overbought reversal setup\n"+
				"• Stochastic confirmation\n"+
				"• Volume and trend support\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Reward Amount: $%.2f\n"+
				"• Position Value: $%.2f",
				latestPrice,
				latestPrice+stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice-takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				takeProfitDistance/stopLossDistance,
				leverage,
				positionSize*100/accountSize,
				latestRSI,
				prevRSI,
				latestSlowK,
				latestSlowD,
				latestEMA20,
				latestEMA50,
				volumeStrength,
				priceVsEMA20,
				volatilityRatio,
				atrValue,
				accountSize,
				riskAmount,
				rewardAmount,
				positionSize,
			),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}
