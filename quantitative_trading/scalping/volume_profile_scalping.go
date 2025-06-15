package scalping

import (
	"fmt"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

type VolumeProfileScalpingStrategy struct {
	name        string
	description string
}

func NewVolumeProfileScalpingStrategy() *VolumeProfileScalpingStrategy {
	return &VolumeProfileScalpingStrategy{
		name:        "Volume Profile Scalping",
		description: "Scalping strategy based on volume profile analysis and price levels",
	}
}

func (s *VolumeProfileScalpingStrategy) GetName() string {
	return s.name
}

func (s *VolumeProfileScalpingStrategy) GetDescription() string {
	return s.description
}

func (s *VolumeProfileScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketRanging, common.MarketSideways,
		common.MarketLowVolatility, common.MarketSqueeze,
		common.MarketAccumulation, common.MarketDistribution:
		return true
	default:
		return false
	}
}

// VolumeProfileScalping implements a scalping strategy based on volume profile analysis
func VolumeProfileScalping(candles5m []repository.Candle) (*strategies.Signal, error) {
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

	// Calculate RSI for additional confirmation
	rsi := talib.Rsi(closes, 14)

	// Calculate VWAP
	vwap := calculateVWAP(closes, volumes)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	latestRSI := rsi[len(rsi)-1]
	latestVWAP := vwap[len(vwap)-1]

	// Calculate market metrics
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	priceVsVWAP := ((latestPrice - latestVWAP) / latestVWAP) * 100
	priceVsEMA20 := ((latestPrice - latestEMA20) / latestEMA20) * 100
	volatilityPercent := (atrValue / latestPrice) * 100

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestPrice > latestVWAP && latestEMA20 > latestEMA50 {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else if latestPrice < latestVWAP && latestEMA20 < latestEMA50 {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on volume profile
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on volume profile
	var expectedMove float64

	// Volume strength
	if volumeStrength > 200.0 {
		expectedMove = 0.7 // Strong volume, expect 0.7% move
	} else if volumeStrength > 150.0 {
		expectedMove = 0.5 // Moderate volume, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak volume, expect 0.3% move
	}

	// Trend confirmation
	if latestEMA20 > latestEMA50 {
		expectedMove *= 1.2 // Uptrend confirmation
	} else if latestEMA20 < latestEMA50 {
		expectedMove *= 1.2 // Downtrend confirmation
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
	if volatilityPercent > 2.0 {
		leverage *= 0.5 // Reduce leverage in high volatility
	} else if volatilityPercent > 1.0 {
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

	// Trading logic
	if latestPrice > latestVWAP && // Price above VWAP
		volumeStrength > 150 && // Strong volume
		latestRSI < 60 && // Not overbought
		priceVsEMA20 > -1.0 && // Price near or above EMA20
		latestEMA20 > latestEMA50 { // Uptrend confirmation

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Volume Profile - BUY Signal %s/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• VWAP: %.5f\n"+
				"• Price vs VWAP: %.2f%%\n"+
				"• Volume Strength: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Volume profile breakout setup\n"+
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
				latestVWAP,
				priceVsVWAP,
				volumeStrength,
				latestRSI,
				latestEMA20,
				latestEMA50,
				priceVsEMA20,
				atrValue,
				volatilityPercent,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestPrice < latestVWAP && // Price below VWAP
		volumeStrength > 150 && // Strong volume
		latestRSI > 40 && // Not oversold
		priceVsEMA20 < 1.0 && // Price near or below EMA20
		latestEMA20 < latestEMA50 { // Downtrend confirmation

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Volume Profile - SELL Signal %s/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• VWAP: %.5f\n"+
				"• Price vs VWAP: %.2f%%\n"+
				"• Volume Strength: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Volume profile breakdown setup\n"+
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
				latestVWAP,
				priceVsVWAP,
				volumeStrength,
				latestRSI,
				latestEMA20,
				latestEMA50,
				priceVsEMA20,
				atrValue,
				volatilityPercent,
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

func (s *VolumeProfileScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	return VolumeProfileScalping(candles["5m"])
}
