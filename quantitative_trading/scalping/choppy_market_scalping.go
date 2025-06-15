package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// ChoppyMarketScalpingStrategy is designed for choppy market conditions
type ChoppyMarketScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewChoppyMarketScalpingStrategy() *ChoppyMarketScalpingStrategy {
	return &ChoppyMarketScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Choppy Market Scalping",
		},
	}
}

func (s *ChoppyMarketScalpingStrategy) GetDescription() string {
	return "Scalping strategy using ADX, Stochastic, and Volume Profile for choppy markets. Best for identifying short-term opportunities in erratic price movements."
}

func (s *ChoppyMarketScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketChoppy:
		return true
	default:
		return false
	}
}

func (s *ChoppyMarketScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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

	// Calculate ADX for trend strength
	adx := talib.Adx(highs, lows, closes, 14)
	if len(adx) < 2 {
		return nil, nil
	}

	// Calculate Stochastic with optimized parameters for choppy markets
	slowK, slowD := talib.Stoch(highs, lows, closes, 9, 3, talib.SMA, 3, talib.SMA)
	if len(slowK) < 2 || len(slowD) < 2 {
		return nil, nil
	}

	// Calculate RSI for additional confirmation
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}

	// Calculate Bollinger Bands for volatility and range
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate Volume Profile with EMA for smoother signals
	volumeEMA := talib.Ema(volumes, 20)
	if len(volumeEMA) < 2 {
		return nil, nil
	}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestADX := adx[len(adx)-1]
	latestSlowK := slowK[len(slowK)-1]
	latestSlowD := slowD[len(slowD)-1]
	latestRSI := rsi[len(rsi)-1]
	latestBBUpper := bbUpper[len(bbUpper)-1]
	latestBBMiddle := bbMiddle[len(bbMiddle)-1]
	latestBBLower := bbLower[len(bbLower)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeEMA := volumeEMA[len(volumeEMA)-1]
	latestATR := atr[len(atr)-1]

	// Calculate market structure
	prevHigh := highs[len(highs)-2]
	prevLow := lows[len(lows)-2]
	prevClose := closes[len(closes)-2]

	// Detect price action patterns
	isInsideBar := latestPrice < prevHigh && latestPrice > prevLow
	isOutsideBar := latestPrice > prevHigh && latestPrice < prevLow
	isDoji := math.Abs(latestPrice-prevClose) < (latestATR * 0.1)

	// Calculate volatility ratio
	volatilityRatio := latestATR / latestBBMiddle * 100

	// Calculate volume strength
	volumeStrength := (latestVolume / latestVolumeEMA) * 100

	// Calculate price momentum
	priceMomentum := (latestPrice - prevClose) / prevClose * 100

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if priceMomentum < 0 { // Negative momentum
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on choppy market conditions
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on market conditions
	var expectedMove float64

	// Bollinger Band squeeze strength
	bbWidth := (latestBBUpper - latestBBLower) / latestBBMiddle * 100
	if bbWidth < 1.0 {
		expectedMove = 0.7 // Strong squeeze, expect 0.7% move
	} else if bbWidth < 2.0 {
		expectedMove = 0.5 // Moderate squeeze, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak squeeze, expect 0.3% move
	}

	// Price momentum confirmation
	if math.Abs(priceMomentum) > 0.5 {
		expectedMove *= 1.5 // Strong momentum confirms move
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
	if volatilityRatio > 2.0 {
		leverage *= 0.5 // Reduce leverage in high volatility
	} else if volatilityRatio > 1.0 {
		leverage *= 0.7 // Moderate reduction in medium volatility
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
	}

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Calculate actual risk:reward ratio
	riskRewardRatio := takeProfitDistance / stopLossDistance

	// Calculate position size based on risk
	accountSize := 1000.0
	accountRisk := 0.02
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (stopLossDistance / latestPrice)

	// Calculate signal confidence
	signalConfidence := 100.0 - riskPercent

	// Trading logic with improved technical analysis
	if latestADX < 25 && // Weak trend
		latestSlowK < 20 && latestSlowD < 20 && // Oversold
		latestRSI < 30 && // RSI oversold
		latestPrice < latestBBLower && // Price below lower BB
		volumeStrength > 120 && // Above average volume
		!isDoji && // Not a doji
		priceMomentum < 0 { // Negative momentum

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Choppy Market Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• ADX: %.2f (Weak Trend)\n"+
				"• Stochastic K: %.2f (Oversold)\n"+
				"• Stochastic D: %.2f (Oversold)\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• BB Lower: %.5f\n"+
				"• BB Middle: %.5f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n"+
				"• Inside Bar: %v\n"+
				"• Outside Bar: %v\n"+
				"• Doji: %v\n\n"+
				"💡 Trade Notes:\n"+
				"• Multiple oversold indicators\n"+
				"• Price below lower BB\n"+
				"• Strong volume confirmation\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Expected Move: %.2f%%",
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				positionSize*100/accountSize,
				signalConfidence,
				latestADX,
				latestSlowK,
				latestSlowD,
				latestRSI,
				latestBBLower,
				latestBBMiddle,
				latestATR,
				volatilityRatio,
				volumeStrength,
				priceMomentum,
				isInsideBar,
				isOutsideBar,
				isDoji,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestADX < 25 && // Weak trend
		latestSlowK > 80 && latestSlowD > 80 && // Overbought
		latestRSI > 70 && // RSI overbought
		latestPrice > latestBBUpper && // Price above upper BB
		volumeStrength > 120 && // Above average volume
		!isDoji && // Not a doji
		priceMomentum > 0 { // Positive momentum

		// Calculate stop loss and take profit based on technical levels
		stopLossDistance := math.Max(latestATR*1.5, (latestBBUpper-latestPrice)*1.2)
		takeProfitDistance := math.Max(latestATR*2.5, (latestPrice-latestBBMiddle)*1.5)

		// Calculate position size based on risk
		accountSize := 1000.0
		accountRisk := 0.02
		riskAmount := accountSize * accountRisk
		positionSize := riskAmount / (stopLossDistance / latestPrice)

		// Calculate leverage based on volatility
		leverage := 10.0
		if volatilityRatio < 1.0 {
			leverage = 20.0
		} else if volatilityRatio < 2.0 {
			leverage = 15.0
		} else if volatilityRatio < 3.0 {
			leverage = 10.0
		} else {
			leverage = 5.0
		}

		// Adjust leverage based on market structure
		if isInsideBar {
			leverage *= 0.8 // Reduce leverage for inside bars
		}

		// Cap maximum leverage
		if leverage > 20.0 {
			leverage = 20.0
		}

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Choppy Market Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• ADX: %.2f (Weak Trend)\n"+
				"• Stochastic K: %.2f (Overbought)\n"+
				"• Stochastic D: %.2f (Overbought)\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• BB Upper: %.5f\n"+
				"• BB Middle: %.5f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n"+
				"• Inside Bar: %v\n"+
				"• Outside Bar: %v\n"+
				"• Doji: %v\n\n"+
				"💡 Trade Notes:\n"+
				"• Multiple overbought indicators\n"+
				"• Price above upper BB\n"+
				"• Strong volume confirmation\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Expected Move: %.2f%%",
				latestPrice,
				latestPrice+stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice-takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				takeProfitDistance/stopLossDistance,
				leverage,
				positionSize*100/accountSize,
				latestADX,
				latestSlowK,
				latestSlowD,
				latestRSI,
				latestBBUpper,
				latestBBMiddle,
				latestATR,
				volatilityRatio,
				volumeStrength,
				priceMomentum,
				isInsideBar,
				isOutsideBar,
				isDoji,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}

/*
func findNearestResistance(price float64, highs []float64) float64 {
	nearestResistance := math.Inf(-1)
	for _, high := range highs {
		if high > price && high > nearestResistance {
			nearestResistance = high
		}
	}
	return nearestResistance
}

func findNearestSupport(price float64, lows []float64) float64 {
	nearestSupport := math.Inf(1)
	for _, low := range lows {
		if low < price && low < nearestSupport {
			nearestSupport = low
		}
	}
	return nearestSupport
}
*/
