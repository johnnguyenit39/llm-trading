package backtesting

import (
	"encoding/json"
	"fmt"
	"j-ai-trade/brokers/okx"
	"j-ai-trade/brokers/okx/types"
	"j-ai-trade/modules/order/model"
	"strings"

	"gorm.io/gorm"
)

type BackTesting struct {
	db *gorm.DB
}

func NewBackTesting(db *gorm.DB) *BackTesting {
	return &BackTesting{
		db: db,
	}
}

// ExecuteFuturesOrder executes a futures order and records it in the database
func (b *BackTesting) ExecuteFuturesOrder(symbol string, amount, price float64, decision string, strategy string, takeProfit, stopLoss float64) error {
	// Initialize OKX service
	okxService := okx.GetInstance()

	// Sync time with OKX server before making the request
	if err := okxService.SyncTime(); err != nil {
		return fmt.Errorf("failed to sync time with OKX: %v", err)
	}

	// Create currency pair
	currencyPair := okxService.NewCurrencyPair("BTC", "USDT")

	// Determine order side based on decision
	var side types.OrderSide
	var posSide string
	switch strings.ToLower(decision) {
	case "long", "buy":
		side = types.Buy
		posSide = "long"
	case "short", "sell":
		side = types.Sell
		posSide = "short"
	default:
		return fmt.Errorf("invalid decision: %s", decision)
	}

	// Create futures order
	response, err := okxService.CreateFuturesOrder(
		currencyPair,
		amount,
		price,
		side,
		types.Market, // Using market order type
		10,           // Default leverage
		posSide,
		takeProfit, takeProfit, // Use the same price for trigger and order
		stopLoss, stopLoss, // Use the same price for trigger and order
	)
	if err != nil {
		return fmt.Errorf("failed to create futures order: %v", err)
	}

	// Parse response to get order ID
	var responseMap map[string]interface{}
	if err := json.Unmarshal(response, &responseMap); err != nil {
		return fmt.Errorf("failed to parse order response: %v", err)
	}

	// Extract order ID from response
	data, ok := responseMap["data"].([]interface{})
	if !ok || len(data) == 0 {
		return fmt.Errorf("invalid response data format")
	}

	firstData, ok := data[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data item format")
	}

	orderID, ok := firstData["ordId"].(string)
	if !ok {
		return fmt.Errorf("failed to get order ID from response")
	}

	// Create order record
	order := &model.Order{
		Broker:        "okx",
		BrokerOrderID: orderID,
		Decision:      decision,
		Pair:          currencyPair.Symbol,
		Type:          "futures",
		Entry:         price,
		Strategy:      strategy,
	}

	// Save order to database
	if err := b.db.Create(order).Error; err != nil {
		return fmt.Errorf("failed to save order to database: %v", err)
	}

	return nil
}
