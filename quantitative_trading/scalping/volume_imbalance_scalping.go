package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

type VolumeImbalanceScalpingStrategy struct {
	name        string
	description string
}

func NewVolumeImbalanceScalpingStrategy() *VolumeImbalanceScalpingStrategy {
	return &VolumeImbalanceScalpingStrategy{
		name:        "Volume Imbalance Scalping",
		description: "Scalping strategy based on volume imbalance analysis and order flow",
	}
}

func (s *VolumeImbalanceScalpingStrategy) GetName() string {
	return s.name
}

func (s *VolumeImbalanceScalpingStrategy) GetDescription() string {
	return s.description
}

func (s *VolumeImbalanceScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketHighVolatility, common.MarketVolatile,
		common.MarketBreakout, common.MarketBreakoutUp, common.MarketBreakoutDown,
		common.MarketAccumulation, common.MarketDistribution,
		common.MarketReversal, common.MarketReversalUp, common.MarketReversalDown:
		return true
	default:
		return false
	}
}

func (s *VolumeImbalanceScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	return VolumeImbalanceScalping(candles["5m"])
}

// VolumeImbalanceScalping implements a scalping strategy based on volume imbalance analysis
func VolumeImbalanceScalping(candles5m []repository.Candle) (*strategies.Signal, error) {
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

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestVolumeEMA := volumeEMA[len(volumeEMA)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	latestRSI := rsi[len(rsi)-1]

	// Calculate market metrics
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	volumeTrend := (latestVolumeEMA / latestVolumeMA) * 100
	priceVsEMA20 := ((latestPrice - latestEMA20) / latestEMA20) * 100
	volatilityRatio := atrValue / latestPrice * 100

	// Calculate volume imbalance
	volumeImbalance := calculateVolumeImbalance(closes, volumes, 5)

	// Calculate leverage based on volatility and volume imbalance
	leverage := 3.0 // Default for volume imbalance trading
	if volatilityRatio > 3.0 {
		leverage = 2.0
	} else if volatilityRatio > 2.0 {
		leverage = 2.5
	} else if volatilityRatio > 1.0 {
		leverage = 3.0
	} else {
		leverage = 4.0
	}

	// Adjust leverage based on volume imbalance
	if math.Abs(volumeImbalance) > 2.0 {
		leverage *= 0.8 // Reduce leverage on extreme volume imbalance
	}

	// Cap maximum leverage
	if leverage > 5.0 {
		leverage = 5.0
	}

	// Calculate stop loss and take profit based on volume imbalance
	stopLossDistance := math.Max(atrValue*1.2, latestPrice*0.01) // Minimum 1% stop loss
	takeProfitDistance := math.Max(atrValue*2.0, stopLossDistance*1.5)

	// Calculate position size based on risk
	accountSize := 1000.0
	accountRisk := 0.02
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (stopLossDistance / latestPrice)
	rewardAmount := riskAmount * (takeProfitDistance / stopLossDistance)

	// Trading logic with improved volume imbalance analysis
	if volumeImbalance > 1.5 && // Strong buying pressure
		volumeStrength > 150 && // Above average volume
		volumeTrend > 110 && // Increasing volume trend
		latestRSI < 60 && // Not overbought
		priceVsEMA20 > -1.0 && // Price near or above EMA20
		latestEMA20 > latestEMA50 { // Uptrend confirmation

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Volume Imbalance - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• Volume Imbalance: %.2f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Volume Trend: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• Volume imbalance breakout setup\n"+
				"• Strong buying pressure\n"+
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
				volumeImbalance,
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
	} else if volumeImbalance < -1.5 && // Strong selling pressure
		volumeStrength > 150 && // Above average volume
		volumeTrend > 110 && // Increasing volume trend
		latestRSI > 40 && // Not oversold
		priceVsEMA20 < 1.0 && // Price near or below EMA20
		latestEMA20 < latestEMA50 { // Downtrend confirmation

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Volume Imbalance - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• Volume Imbalance: %.2f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Volume Trend: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• Volume imbalance breakdown setup\n"+
				"• Strong selling pressure\n"+
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
				volumeImbalance,
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

// Helper function to calculate volume imbalance
func calculateVolumeImbalance(prices, volumes []float64, lookback int) float64 {
	if len(prices) < lookback || len(volumes) < lookback {
		return 0
	}

	var buyVolume, sellVolume float64
	for i := len(prices) - lookback; i < len(prices); i++ {
		if prices[i] > prices[i-1] {
			buyVolume += volumes[i]
		} else if prices[i] < prices[i-1] {
			sellVolume += volumes[i]
		}
	}

	if sellVolume == 0 {
		return 2.0 // Maximum imbalance
	}

	return buyVolume / sellVolume
}
