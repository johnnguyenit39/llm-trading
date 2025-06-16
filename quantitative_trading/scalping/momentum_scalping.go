package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// MomentumScalping implements a scalping strategy based on price momentum
func MomentumScalping(candles5m []repository.BinanceCandle) (*strategies.Signal, error) {
	// Convert candle data to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, candle := range candles5m {
		closes[i] = candle.Close
		volumes[i] = candle.Volume
		highs[i] = candle.High
		lows[i] = candle.Low
	}

	// Calculate RSI
	rsi := talib.Rsi(closes, 14)

	// Calculate ROC (Rate of Change)
	roc := talib.Roc(closes, 10)

	// Calculate MACD for trend confirmation
	macd, signal, hist := talib.Macd(closes, 12, 26, 9)

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	atrValue := atr[len(atr)-1]

	// Calculate EMA for trend direction
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)

	// Calculate Volume Profile with EMA
	volumeEMA := talib.Ema(volumes, 20)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestRSI := rsi[len(rsi)-1]
	latestROC := roc[len(roc)-1]
	latestMACD := macd[len(macd)-1]
	latestSignal := signal[len(signal)-1]
	latestHist := hist[len(hist)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeEMA := volumeEMA[len(volumeEMA)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestRSI < 30 && latestROC > 0 && latestMACD > latestSignal && latestHist > 0 {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else if latestRSI > 70 && latestROC < 0 && latestMACD < latestSignal && latestHist < 0 {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on momentum strength
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on momentum conditions
	var expectedMove float64

	// Momentum indicators
	priceMomentum := (latestPrice - closes[len(closes)-2]) / closes[len(closes)-2] * 100
	volumeStrength := (latestVolume / latestVolumeEMA) * 100
	macdMomentum := latestMACD - macd[len(macd)-2]

	// Calculate expected move based on momentum strength
	if math.Abs(priceMomentum) > 1.0 {
		expectedMove = 0.7 // Strong momentum, expect 0.7% move
	} else if math.Abs(priceMomentum) > 0.5 {
		expectedMove = 0.5 // Moderate momentum, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak momentum, expect 0.3% move
	}

	// Volume confirmation
	if volumeStrength > 150.0 {
		expectedMove *= 1.5 // Strong volume confirms move
	} else if volumeStrength > 120.0 {
		expectedMove *= 1.2 // Above average volume confirms move
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
	volatilityRatio := atrValue / latestPrice * 100
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
	accountSize := 1000.0 // $1000 account
	accountRisk := 0.02   // 2% risk per trade
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (riskPercent / 100.0)

	// Calculate signal confidence
	signalConfidence := 100.0 - riskPercent

	// Trading logic with improved momentum analysis
	if latestRSI < 30 && // Oversold
		latestROC > 0 && // Positive momentum
		latestMACD > latestSignal && // MACD above signal
		latestHist > 0 && // Positive histogram
		latestPrice > latestEMA20 && // Price above short-term trend
		latestEMA20 > latestEMA50 && // Uptrend confirmation
		volumeStrength > 150 && // Strong volume
		priceMomentum > 0 { // Positive price momentum

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Momentum Scalping - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• ROC: %.2f%%\n"+
				"• MACD: %.6f\n"+
				"• Signal: %.6f\n"+
				"• Histogram: %.6f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n"+
				"• MACD Momentum: %.6f\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong momentum setup\n"+
				"• Multiple trend confirmations\n"+
				"• High volume confirmation\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Expected Move: %.2f%%",
				candles5m[len(candles5m)-1].Symbol,
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				positionSize*100/accountSize,
				signalConfidence,
				latestRSI,
				latestROC,
				latestMACD,
				latestSignal,
				latestHist,
				latestEMA20,
				latestEMA50,
				volumeStrength,
				priceMomentum,
				macdMomentum,
				volatilityRatio,
				atrValue,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestRSI > 70 && // Overbought
		latestROC < 0 && // Negative momentum
		latestMACD < latestSignal && // MACD below signal
		latestHist < 0 && // Negative histogram
		latestPrice < latestEMA20 && // Price below short-term trend
		latestEMA20 < latestEMA50 && // Downtrend confirmation
		volumeStrength > 150 && // Strong volume
		priceMomentum < 0 { // Negative price momentum

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Momentum Scalping - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• ROC: %.2f%%\n"+
				"• MACD: %.6f\n"+
				"• Signal: %.6f\n"+
				"• Histogram: %.6f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n"+
				"• MACD Momentum: %.6f\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong momentum setup\n"+
				"• Multiple trend confirmations\n"+
				"• High volume confirmation\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Expected Move: %.2f%%",
				candles5m[len(candles5m)-1].Symbol,
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				positionSize*100/accountSize,
				signalConfidence,
				latestRSI,
				latestROC,
				latestMACD,
				latestSignal,
				latestHist,
				latestEMA20,
				latestEMA50,
				volumeStrength,
				priceMomentum,
				macdMomentum,
				volatilityRatio,
				atrValue,
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
