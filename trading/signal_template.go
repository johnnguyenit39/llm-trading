package trading

import (
	"fmt"
	"math"
	"strings"
)

// ==== Base Signal Models ====

type BaseSignalModel struct {
	Symbol     string  `json:"symbol"`
	Side       string  `json:"side"`
	Entry      float64 `json:"entry"`
	TakeProfit float64 `json:"take_profit"`
	StopLoss   float64 `json:"stop_loss"`
	Leverage   float64 `json:"leverage"`
	AmountUSD  float64 `json:"amount_usd"`
	ATRPercent float64 `json:"atr_percent"`
}

type EnhancedSignalModel struct {
	BaseSignalModel
	TrailingStop      float64 `json:"trailing_stop"`
	PositionSize      float64 `json:"position_size"`
	RiskAmount        float64 `json:"risk_amount"`
	RiskPercent       float64 `json:"risk_percent"`
	MaxHoldTime       int     `json:"max_hold_time"`
	ProfitLockTime    int     `json:"profit_lock_time"`
	UseTrailingStop   bool    `json:"use_trailing_stop"`
	DrawdownProtected bool    `json:"drawdown_protected"`
}

// ==== Signal Generation Template ====

type SignalTemplate struct {
	StrategyName string
	Icon         string
	Description  string
}

func NewSignalTemplate(strategyName string) *SignalTemplate {
	return &SignalTemplate{
		StrategyName: strategyName,
		Icon:         "📊",
		Description:  "",
	}
}

// Generate base signal string
func (t *SignalTemplate) GenerateBaseSignal(symbol, side string, entry, leverage, atrPercent float64, marginUSD float64) string {
	icon := getSignalIcon(side)

	result := fmt.Sprintf("%s %s Signal: %s\n", icon, t.StrategyName, strings.ToUpper(side))
	result += fmt.Sprintf("Symbol: %s\n", strings.ToUpper(symbol))
	result += fmt.Sprintf("Entry: %.4f\n", entry)
	result += fmt.Sprintf("Leverage: %.1fx\n", leverage)
	result += fmt.Sprintf("ATR%%(20): %.4f\n", atrPercent)
	result += fmt.Sprintf("Simulated Fund: $%.1f USD\n\n", marginUSD)

	return result
}

// Generate SL/TP section
func (t *SignalTemplate) GenerateSLTPSection(sl, tp float64) string {
	result := fmt.Sprintf("Stop Loss: %.4f\n", sl)
	result += fmt.Sprintf("Take Profit: %.4f\n\n", tp)
	return result
}

// Generate profit/loss calculation
func (t *SignalTemplate) GenerateProfitLossSection(entry, sl, tp, leverage, marginUSD float64) string {
	positionSize := marginUSD * leverage
	slDistance := math.Abs(entry - sl)
	slPercent := (slDistance / entry) * 100
	tpDistance := math.Abs(tp - entry)
	tpPercent := (tpDistance / entry) * 100

	potentialLoss := positionSize * (slPercent / 100)
	potentialProfit := positionSize * (tpPercent / 100)

	balanceAfterLoss := marginUSD - potentialLoss
	balanceAfterWin := marginUSD + potentialProfit

	result := fmt.Sprintf("Position Size: $%.1f\n", positionSize)
	result += fmt.Sprintf("Potential Loss: $%.2f (%.2f%%) - Balance after loss: $%.2f\n",
		potentialLoss, slPercent, balanceAfterLoss)
	result += fmt.Sprintf("Potential Profit: $%.2f (%.2f%%) - Balance after win: $%.2f\n",
		potentialProfit, tpPercent, balanceAfterWin)

	return result
}

// Generate risk management section
func (t *SignalTemplate) GenerateRiskManagementSection(signal EnhancedSignalModel, accountBalance float64) string {
	result := "\n=== RISK MANAGEMENT ===\n"
	result += fmt.Sprintf("Position Size: $%.2f\n", signal.PositionSize)
	result += fmt.Sprintf("Risk Amount: $%.2f (%.2f%% of account)\n", signal.RiskAmount, signal.RiskPercent)

	if signal.UseTrailingStop {
		result += fmt.Sprintf("Trailing Stop: %.4f (activated at %.1f%% profit)\n", signal.TrailingStop, 0.5)
	} else {
		result += "Trailing Stop: Not activated yet\n"
	}

	result += fmt.Sprintf("Max Hold Time: %d minutes\n", signal.MaxHoldTime)
	result += fmt.Sprintf("Profit Lock Time: %d minutes\n", signal.ProfitLockTime)
	result += fmt.Sprintf("Max Drawdown: %.1f%%\n", 15.0)
	result += fmt.Sprintf("Daily Loss Limit: %.1f%%\n", 10.0)
	result += fmt.Sprintf("Account Balance: $%.2f\n", accountBalance)
	result += "Drawdown Protection: ✅ Active\n"

	return result
}

// Generate quality score section
func (t *SignalTemplate) GenerateQualityScoreSection(qualityScore *SignalQualityScore) string {
	result := "\n=== SIGNAL QUALITY ANALYSIS ===\n"
	result += fmt.Sprintf("Overall Score: %.1f/10\n", qualityScore.overallScore)
	result += fmt.Sprintf("Trend Score: %.1f/10\n", qualityScore.trendScore)
	result += fmt.Sprintf("Pattern Score: %.1f/10\n", qualityScore.patternScore)
	result += fmt.Sprintf("Volume Score: %.1f/10\n", qualityScore.volumeScore)
	result += fmt.Sprintf("Market Score: %.1f/10\n", qualityScore.marketScore)
	result += fmt.Sprintf("Confirmation Score: %.1f/10\n", qualityScore.confirmationScore)

	// Add quality assessment
	if qualityScore.overallScore >= 8.5 {
		result += "Quality Assessment: 🟢 EXCELLENT\n"
	} else if qualityScore.overallScore >= 7.5 {
		result += "Quality Assessment: 🟡 GOOD\n"
	} else if qualityScore.overallScore >= 7.0 {
		result += "Quality Assessment: 🟠 ACCEPTABLE\n"
	} else {
		result += "Quality Assessment: 🔴 POOR\n"
	}

	return result
}

// Generate complete signal string
func (t *SignalTemplate) GenerateCompleteSignal(
	symbol, side string,
	entry, sl, tp, leverage, atrPercent, marginUSD float64,
	enhancedSignal *EnhancedSignalModel,
	qualityScore *SignalQualityScore,
	accountBalance float64,
) string {

	// Base signal
	result := t.GenerateBaseSignal(symbol, side, entry, leverage, atrPercent, marginUSD)

	// SL/TP section
	result += t.GenerateSLTPSection(sl, tp)

	// Profit/Loss section
	result += t.GenerateProfitLossSection(entry, sl, tp, leverage, marginUSD)

	// Risk management (if enhanced signal provided)
	if enhancedSignal != nil {
		result += t.GenerateRiskManagementSection(*enhancedSignal, accountBalance)
	}

	// Quality score (if provided)
	if qualityScore != nil {
		result += t.GenerateQualityScoreSection(qualityScore)
	}

	return strings.TrimSpace(result)
}

// ==== Utility Functions ====

// Note: getSignalIcon, SignalQualityScore, and constants are already defined in scalping_1.go
// This template uses the shared types and constants from other strategy files
