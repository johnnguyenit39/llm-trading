package scalping

import (
	"fmt"
	"math"
	"time"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// OrderbookTapeScalping implements a scalping strategy based on orderbook tape analysis
func OrderbookTapeScalping(candles []repository.Candle) (*strategies.Signal, error) {
	if len(candles) < 20 {
		return nil, nil
	}

	// Convert candles to float64 arrays
	closes := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	volumes := make([]float64, len(candles))

	for i, candle := range candles {
		closes[i] = candle.Close
		highs[i] = candle.High
		lows[i] = candle.Low
		volumes[i] = candle.Volume
	}

	// Calculate indicators
	atr := talib.Atr(highs, lows, closes, 14)
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)
	rsi := talib.Rsi(closes, 14)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestATR := atr[len(atr)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	latestRSI := rsi[len(rsi)-1]

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestPrice > latestEMA20 && latestEMA20 > latestEMA50 && latestRSI < 70 {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else if latestPrice < latestEMA20 && latestEMA20 < latestEMA50 && latestRSI > 30 {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on market metrics
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on market conditions
	var expectedMove float64

	// Market metrics
	volumeStrength := latestVolume / talib.Sma(volumes, 20)[len(volumes)-1]
	volatilityRatio := latestATR / talib.Sma(atr, 20)[len(atr)-1]
	trendStrength := math.Abs(latestEMA20-latestEMA50) / latestATR

	// Calculate expected move based on trend strength
	if trendStrength > 0.5 {
		expectedMove = 0.7 // Strong trend, expect 0.7% move
	} else if trendStrength > 0.3 {
		expectedMove = 0.5 // Moderate trend, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak trend, expect 0.3% move
	}

	// Volume confirmation
	if volumeStrength > 1.5 {
		expectedMove *= 1.5 // Strong volume confirms move
	} else if volumeStrength > 1.2 {
		expectedMove *= 1.2 // Above average volume confirms move
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
	volatilityPercent := (latestATR / latestPrice) * 100
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
	if latestPrice > latestEMA20 && latestEMA20 > latestEMA50 && latestRSI < 70 {
		// Strong buy signal
		if volumeStrength > 1.2 && trendStrength > 0.5 && volatilityRatio < 1.5 {
			return &strategies.Signal{
				Type:  "BUY",
				Price: latestPrice,
				Time:  time.Now(),
				Description: fmt.Sprintf("🚀 Orderbook Tape Scalping - BUY Signal %s/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (-%.2f%%)\n"+
					"• Take Profit: %.5f (+%.2f%%)\n"+
					"• Risk/Reward: 1:%.2f\n"+
					"• Leverage: %.1fx\n"+
					"• Position Size: %.2f%% of account\n"+
					"• Signal Confidence: %.1f%%\n\n"+
					"📈 Technical Analysis:\n"+
					"• EMA20: %.5f\n"+
					"• EMA50: %.5f\n"+
					"• RSI: %.2f\n"+
					"• Volume Strength: %.2f\n"+
					"• Trend Strength: %.2f\n"+
					"• Volatility Ratio: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Trade Notes:\n"+
					"• Strong trend following setup\n"+
					"• Max risk per trade: 2%%\n"+
					"• Account Size: $%.2f\n"+
					"• Risk Amount: $%.2f\n"+
					"• Expected Move: %.2f%%",
					candles[len(candles)-1].Symbol,
					latestPrice,
					latestPrice-stopLossDistance,
					riskPercent,
					latestPrice+takeProfitDistance,
					rewardPercent,
					riskRewardRatio,
					leverage,
					positionSize*100/accountSize,
					signalConfidence,
					latestEMA20,
					latestEMA50,
					latestRSI,
					volumeStrength,
					trendStrength,
					volatilityRatio,
					latestATR,
					volatilityPercent,
					accountSize,
					riskAmount,
					expectedMove,
				),
				StopLoss:   latestPrice - stopLossDistance,
				TakeProfit: latestPrice + takeProfitDistance,
				Leverage:   leverage,
			}, nil
		}
	} else if latestPrice < latestEMA20 && latestEMA20 < latestEMA50 && latestRSI > 30 {
		// Strong sell signal
		if volumeStrength > 1.2 && trendStrength > 0.5 && volatilityRatio < 1.5 {
			return &strategies.Signal{
				Type:  "SELL",
				Price: latestPrice,
				Time:  time.Now(),
				Description: fmt.Sprintf("🔻 Orderbook Tape Scalping - SELL Signal %s/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (+%.2f%%)\n"+
					"• Take Profit: %.5f (-%.2f%%)\n"+
					"• Risk/Reward: 1:%.2f\n"+
					"• Leverage: %.1fx\n"+
					"• Position Size: %.2f%% of account\n"+
					"• Signal Confidence: %.1f%%\n\n"+
					"📈 Technical Analysis:\n"+
					"• EMA20: %.5f\n"+
					"• EMA50: %.5f\n"+
					"• RSI: %.2f\n"+
					"• Volume Strength: %.2f\n"+
					"• Trend Strength: %.2f\n"+
					"• Volatility Ratio: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Trade Notes:\n"+
					"• Strong trend following setup\n"+
					"• Max risk per trade: 2%%\n"+
					"• Account Size: $%.2f\n"+
					"• Risk Amount: $%.2f\n"+
					"• Expected Move: %.2f%%",
					candles[len(candles)-1].Symbol,
					latestPrice,
					latestPrice+stopLossDistance,
					riskPercent,
					latestPrice-takeProfitDistance,
					rewardPercent,
					riskRewardRatio,
					leverage,
					positionSize*100/accountSize,
					signalConfidence,
					latestEMA20,
					latestEMA50,
					latestRSI,
					volumeStrength,
					trendStrength,
					volatilityRatio,
					latestATR,
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
	}

	return nil, nil
}

type OrderbookTapeScalpingStrategy struct {
	name        string
	description string
}

func NewOrderbookTapeScalpingStrategy() *OrderbookTapeScalpingStrategy {
	return &OrderbookTapeScalpingStrategy{
		name:        "Orderbook Tape Scalping",
		description: "Scalping strategy based on orderbook tape analysis and order flow",
	}
}

func (s *OrderbookTapeScalpingStrategy) GetName() string {
	return s.name
}

func (s *OrderbookTapeScalpingStrategy) GetDescription() string {
	return s.description
}

func (s *OrderbookTapeScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketHighVolatility, common.MarketVolatile,
		common.MarketBreakout, common.MarketBreakoutUp, common.MarketBreakoutDown,
		common.MarketReversal, common.MarketReversalUp, common.MarketReversalDown:
		return true
	default:
		return false
	}
}

func (s *OrderbookTapeScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	return OrderbookTapeScalping(candles["5m"])
}
