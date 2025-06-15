package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

type VWAPScalpingStrategy struct {
	name        string
	description string
}

func NewVWAPScalpingStrategy() *VWAPScalpingStrategy {
	return &VWAPScalpingStrategy{
		name:        "VWAP Scalping",
		description: "Scalping strategy based on VWAP (Volume Weighted Average Price) bounces and crossovers",
	}
}

func (s *VWAPScalpingStrategy) GetName() string {
	return s.name
}

func (s *VWAPScalpingStrategy) GetDescription() string {
	return s.description
}

func (s *VWAPScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketStrongTrendUp, common.MarketStrongTrendDown,
		common.MarketWeakTrendUp, common.MarketWeakTrendDown,
		common.MarketRanging, common.MarketSideways:
		return true
	default:
		return false
	}
}

func (s *VWAPScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	return VWAPScalping(candles["5m"])
}

// VWAPScalping implements a scalping strategy based on VWAP analysis
func VWAPScalping(candles5m []repository.Candle) (*strategies.Signal, error) {
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

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate EMAs for trend
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)
	volumeEMA := talib.Ema(volumes, 20)

	// Calculate RSI for additional confirmation
	rsi := talib.Rsi(closes, 14)

	// Calculate VWAP
	vwap := calculateVWAP(closes, volumes)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestVolumeEMA := volumeEMA[len(volumeEMA)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	latestRSI := rsi[len(rsi)-1]
	latestVWAP := vwap[len(vwap)-1]

	// Calculate market metrics
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	volumeTrend := (latestVolumeEMA / latestVolumeMA) * 100
	priceVsVWAP := ((latestPrice - latestVWAP) / latestVWAP) * 100
	priceVsEMA20 := ((latestPrice - latestEMA20) / latestEMA20) * 100
	volatilityRatio := atrValue / latestPrice * 100

	// Calculate leverage based on volatility and VWAP distance
	leverage := 3.0 // Default for VWAP trading
	if volatilityRatio > 3.0 {
		leverage = 2.0
	} else if volatilityRatio > 2.0 {
		leverage = 2.5
	} else if volatilityRatio > 1.0 {
		leverage = 3.0
	} else {
		leverage = 4.0
	}

	// Adjust leverage based on VWAP distance
	if math.Abs(priceVsVWAP) > 1.0 {
		leverage *= 0.8 // Reduce leverage when price is far from VWAP
	}

	// Cap maximum leverage
	if leverage > 5.0 {
		leverage = 5.0
	}

	// Calculate stop loss and take profit based on VWAP
	stopLossDistance := math.Max(atrValue*1.2, latestPrice*0.01) // Minimum 1% stop loss
	takeProfitDistance := math.Max(atrValue*2.0, stopLossDistance*1.5)

	// Calculate position size based on risk
	accountSize := 1000.0
	accountRisk := 0.02
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (stopLossDistance / latestPrice)
	rewardAmount := riskAmount * (takeProfitDistance / stopLossDistance)

	// Trading logic with improved VWAP analysis
	if latestPrice > latestVWAP && // Price above VWAP
		priceVsVWAP < 0.5 && // Price not too far from VWAP
		volumeStrength > 120 && // Above average volume
		volumeTrend > 110 && // Increasing volume trend
		latestRSI < 60 && // Not overbought
		priceVsEMA20 > -1.0 && // Price near or above EMA20
		latestEMA20 > latestEMA50 { // Uptrend confirmation

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 VWAP Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• VWAP: %.5f\n"+
				"• Price vs VWAP: %.2f%%\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Volume Trend: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• VWAP bounce setup\n"+
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
				latestVWAP,
				priceVsVWAP,
				volumeStrength,
				volumeTrend,
				latestRSI,
				latestEMA20,
				latestEMA50,
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
	} else if latestPrice < latestVWAP && // Price below VWAP
		priceVsVWAP > -0.5 && // Price not too far from VWAP
		volumeStrength > 120 && // Above average volume
		volumeTrend > 110 && // Increasing volume trend
		latestRSI > 40 && // Not oversold
		priceVsEMA20 < 1.0 && // Price near or below EMA20
		latestEMA20 < latestEMA50 { // Downtrend confirmation

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 VWAP Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• VWAP: %.5f\n"+
				"• Price vs VWAP: %.2f%%\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Volume Trend: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• VWAP rejection setup\n"+
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
				latestVWAP,
				priceVsVWAP,
				volumeStrength,
				volumeTrend,
				latestRSI,
				latestEMA20,
				latestEMA50,
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
