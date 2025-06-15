package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// HighVolatilityScalpingStrategy is designed for high volatility market conditions
type HighVolatilityScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewHighVolatilityScalpingStrategy() *HighVolatilityScalpingStrategy {
	return &HighVolatilityScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "High Volatility Scalping",
		},
	}
}

func (s *HighVolatilityScalpingStrategy) GetDescription() string {
	return "Scalping strategy using ATR, Keltner Channels, and Volume analysis for high volatility markets. Best for extreme market conditions with large price swings."
}

func (s *HighVolatilityScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketHighVolatility:
		return true
	default:
		return false
	}
}

func (s *HighVolatilityScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
		volumes[i] = c.Volume
	}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Calculate EMA for Keltner
	ema := talib.Ema(closes, 20)
	if len(ema) < 2 {
		return nil, nil
	}

	// Calculate Keltner Channels with dynamic multiplier based on volatility
	kcMultiplier := 3.0
	kcUpper := make([]float64, len(ema))
	kcLower := make([]float64, len(ema))
	for i := range ema {
		kcUpper[i] = ema[i] + (kcMultiplier * atr[i])
		kcLower[i] = ema[i] - (kcMultiplier * atr[i])
	}

	// Calculate RSI for overbought/oversold
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}

	// Calculate MACD for trend confirmation
	macd, signal, hist := talib.Macd(closes, 12, 26, 9)
	if len(macd) < 2 {
		return nil, nil
	}

	// Calculate Volume Profile with EMA
	volumeEMA := talib.Ema(volumes, 20)
	if len(volumeEMA) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestATR := atr[len(atr)-1]
	latestEMA := ema[len(ema)-1]
	latestKCUpper := kcUpper[len(kcUpper)-1]
	latestKCLower := kcLower[len(kcLower)-1]
	latestRSI := rsi[len(rsi)-1]
	latestMACD := macd[len(macd)-1]
	latestSignal := signal[len(signal)-1]
	latestHist := hist[len(hist)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeEMA := volumeEMA[len(volumeEMA)-1]

	// Calculate market structure
	prevHigh := highs[len(highs)-2]
	prevLow := lows[len(lows)-2]
	prevClose := closes[len(closes)-2]

	// Detect volatility patterns
	volatilityRatio := latestATR / latestEMA * 100
	priceRange := (prevHigh - prevLow) / prevLow * 100
	isVolatilityExpanding := latestATR > atr[len(atr)-2]
	isVolatilityContraction := latestATR < atr[len(atr)-2]

	// Calculate momentum indicators
	priceMomentum := (latestPrice - prevClose) / prevClose * 100
	volumeStrength := (latestVolume / latestVolumeEMA) * 100
	macdMomentum := latestMACD - macd[len(macd)-2]

	// Trading logic with improved technical analysis
	if latestPrice < latestKCLower && // Price below lower Keltner
		latestRSI < 30 && // RSI oversold
		latestMACD < latestSignal && // MACD below signal
		latestHist < 0 && // Negative histogram
		volumeStrength > 150 && // Strong volume
		isVolatilityExpanding && // Volatility expanding
		priceMomentum < 0 { // Negative momentum

		// Calculate stop loss and take profit based on volatility
		stopLossDistance := math.Max(latestATR*2.0, (latestPrice-latestKCLower)*1.5)
		takeProfitDistance := math.Max(latestATR*3.0, (latestEMA-latestPrice)*2.0)

		// Calculate position size based on risk
		accountSize := 1000.0
		accountRisk := 0.02
		riskAmount := accountSize * accountRisk
		positionSize := riskAmount / (stopLossDistance / latestPrice)
		rewardAmount := riskAmount * (takeProfitDistance / stopLossDistance)

		// Calculate leverage based on volatility
		leverage := 5.0 // Conservative default for high volatility
		if volatilityRatio < 2.0 {
			leverage = 10.0
		} else if volatilityRatio < 3.0 {
			leverage = 7.0
		} else if volatilityRatio < 4.0 {
			leverage = 5.0
		} else {
			leverage = 3.0 // Very conservative for extreme volatility
		}

		// Adjust leverage based on market conditions
		if isVolatilityContraction {
			leverage *= 1.2 // Increase leverage during volatility contraction
		} else if isVolatilityExpanding {
			leverage *= 0.8 // Decrease leverage during volatility expansion
		}

		// Cap maximum leverage
		if leverage > 20.0 {
			leverage = 20.0
		}

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 High Volatility Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• Price below lower Keltner: %.5f\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• MACD: %.6f\n"+
				"• Signal: %.6f\n"+
				"• Histogram: %.6f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• Price Range: %.2f%%\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n"+
				"• MACD Momentum: %.6f\n"+
				"• Volatility Expanding: %v\n"+
				"• Volatility Contraction: %v\n\n"+
				"💡 Trade Notes:\n"+
				"• Multiple oversold indicators\n"+
				"• Strong volume confirmation\n"+
				"• Volatility-based position sizing\n"+
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
				latestKCLower,
				latestRSI,
				latestMACD,
				latestSignal,
				latestHist,
				latestATR,
				volatilityRatio,
				volatilityRatio,
				priceRange,
				volumeStrength,
				priceMomentum,
				macdMomentum,
				isVolatilityExpanding,
				isVolatilityContraction,
				accountSize,
				riskAmount,
				rewardAmount,
				positionSize,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestPrice > latestKCUpper && // Price above upper Keltner
		latestRSI > 70 && // RSI overbought
		latestMACD > latestSignal && // MACD above signal
		latestHist > 0 && // Positive histogram
		volumeStrength > 150 && // Strong volume
		isVolatilityExpanding && // Volatility expanding
		priceMomentum > 0 { // Positive momentum

		// Calculate stop loss and take profit based on volatility
		stopLossDistance := math.Max(latestATR*2.0, (latestKCUpper-latestPrice)*1.5)
		takeProfitDistance := math.Max(latestATR*3.0, (latestPrice-latestEMA)*2.0)

		// Calculate position size based on risk
		accountSize := 1000.0
		accountRisk := 0.02
		riskAmount := accountSize * accountRisk
		positionSize := riskAmount / (stopLossDistance / latestPrice)
		rewardAmount := riskAmount * (takeProfitDistance / stopLossDistance)

		// Calculate leverage based on volatility
		leverage := 5.0 // Conservative default for high volatility
		if volatilityRatio < 2.0 {
			leverage = 10.0
		} else if volatilityRatio < 3.0 {
			leverage = 7.0
		} else if volatilityRatio < 4.0 {
			leverage = 5.0
		} else {
			leverage = 3.0 // Very conservative for extreme volatility
		}

		// Adjust leverage based on market conditions
		if isVolatilityContraction {
			leverage *= 1.2 // Increase leverage during volatility contraction
		} else if isVolatilityExpanding {
			leverage *= 0.8 // Decrease leverage during volatility expansion
		}

		// Cap maximum leverage
		if leverage > 20.0 {
			leverage = 20.0
		}

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 High Volatility Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• Price above upper Keltner: %.5f\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• MACD: %.6f\n"+
				"• Signal: %.6f\n"+
				"• Histogram: %.6f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• Price Range: %.2f%%\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n"+
				"• MACD Momentum: %.6f\n"+
				"• Volatility Expanding: %v\n"+
				"• Volatility Contraction: %v\n\n"+
				"💡 Trade Notes:\n"+
				"• Multiple overbought indicators\n"+
				"• Strong volume confirmation\n"+
				"• Volatility-based position sizing\n"+
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
				latestKCUpper,
				latestRSI,
				latestMACD,
				latestSignal,
				latestHist,
				latestATR,
				volatilityRatio,
				volatilityRatio,
				priceRange,
				volumeStrength,
				priceMomentum,
				macdMomentum,
				isVolatilityExpanding,
				isVolatilityContraction,
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
