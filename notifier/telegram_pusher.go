package notifier

import (
	"context"
	"fmt"

	"j_ai_trade/telegram"
	"j_ai_trade/trading/models"
)

// TelegramPusher renders a TradeDecision as a human-readable message and
// sends it to Telegram. Message formatting lives here so the trading pipeline
// never embeds presentation/transport concerns.
type TelegramPusher struct {
	svc *telegram.TelegramService
}

func NewTelegramPusher(svc *telegram.TelegramService) *TelegramPusher {
	return &TelegramPusher{svc: svc}
}

func (p *TelegramPusher) Push(_ context.Context, d *models.TradeDecision) error {
	if d == nil || d.Direction == models.DirectionNone {
		return nil
	}
	return p.svc.SendMessage(formatDecision(d))
}

func formatDecision(d *models.TradeDecision) string {
	cappedTxt := ""
	if d.CappedBy != "" {
		cappedTxt = fmt.Sprintf(" [capped-by %s]", d.CappedBy)
	}
	return fmt.Sprintf(
		"%s %s [%s / %s]\nTier: %s (%.0f%% size) | Conf: %.1f | NetRR: %.2f\nAgreement: %d/%d eligible (ratio %.2f)\nEntry: %.4f | SL: %.4f | TP: %.4f\nLev %.0fx | Notional $%.2f | Risk $%.2f%s\nWhy: %s",
		d.Symbol, d.Direction, d.Timeframe, d.Regime,
		d.Tier, d.SizeFactor*100, d.Confidence, d.NetRR,
		d.Agreement, d.EligibleCount, d.AgreeRatio,
		d.Entry, d.StopLoss, d.TakeProfit,
		d.Leverage, d.Notional, d.RiskUSD, cappedTxt,
		d.Reason,
	)
}
