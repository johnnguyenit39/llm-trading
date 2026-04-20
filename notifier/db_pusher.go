package notifier

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	orderModel "j_ai_trade/modules/order/model"
	"j_ai_trade/trading/models"
)

// DBSignalPusher persists fired ensemble decisions as rows in the orders
// table, tagged with the StrategyVersion that produced them.
//
// This pusher is a pure subscriber: it knows nothing about the trading
// pipeline, nothing about Telegram, nothing about the cron schedule. It only
// knows how to translate a TradeDecision into an Order row with Status=SIGNAL.
// Swap it out for a SQLite-backed tracker in tests, or compose it alongside
// other pushers via MultiPusher.
type DBSignalPusher struct {
	db                *gorm.DB
	strategyVersionID uuid.UUID
}

// NewDBSignalPusher returns a pusher that writes into the given DB and
// stamps every row with the supplied strategy version id.
func NewDBSignalPusher(db *gorm.DB, strategyVersionID uuid.UUID) *DBSignalPusher {
	return &DBSignalPusher{db: db, strategyVersionID: strategyVersionID}
}

func (p *DBSignalPusher) Push(ctx context.Context, d *models.TradeDecision) error {
	if d == nil || d.Direction == models.DirectionNone {
		return nil
	}
	if p.db == nil {
		return fmt.Errorf("db pusher: nil db")
	}

	votesJSON, err := json.Marshal(d.Votes)
	if err != nil {
		return fmt.Errorf("marshal votes: %w", err)
	}

	versionID := p.strategyVersionID
	order := &orderModel.Order{
		Broker:            "binance",
		Pair:              d.Symbol,
		Type:              "futures",
		Decision:          d.Direction,
		Strategy:          "ensemble",
		Entry:             d.Entry,
		StrategyVersionID: nullableUUID(versionID),
		Timeframe:         string(d.Timeframe),
		Regime:            string(d.Regime),
		StopLoss:          d.StopLoss,
		TakeProfit:        d.TakeProfit,
		Confidence:        d.Confidence,
		Tier:              d.Tier,
		SizeFactor:        d.SizeFactor,
		Quantity:          d.Quantity,
		Notional:          d.Notional,
		Leverage:          d.Leverage,
		RiskUSD:           d.RiskUSD,
		NetRR:             d.NetRR,
		Agreement:         d.Agreement,
		Eligible:          d.EligibleCount,
		AgreeRatio:        d.AgreeRatio,
		CappedBy:          d.CappedBy,
		Votes:             datatypes.JSON(votesJSON),
		Reason:            d.Reason,
		Status:            orderModel.StatusSignal,
	}
	return p.db.WithContext(ctx).Create(order).Error
}

func nullableUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
