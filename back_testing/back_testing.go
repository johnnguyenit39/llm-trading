package backtesting

import (
	"encoding/json"
	"errors"
	"fmt"
	"j_ai_trade/brokers/okx"
	okxmodel "j_ai_trade/brokers/okx/model"
	"j_ai_trade/brokers/okx/types"
	"strings"

	"gorm.io/gorm"
)

type BackTesting struct {
}

func NewBackTesting(db *gorm.DB) *BackTesting {
	return &BackTesting{}
}

// ExecuteFuturesOrder executes a futures order and records it in the database
func (b *BackTesting) ExecuteFuturesOrder(symbol string, amount, price float64, decision string, strategy string, takeProfit, stopLoss float64, keys *okxmodel.OkxApiKeysModel) error {
	// Initialize OKX service
	okxService := okx.NewOKXService(keys) // Using nil to use environment variables

	parts := strings.Split(symbol, "/")

	if len(parts) != 2 {
		return errors.New("invalid symbol")
	}

	// Sync time with OKX server before making the request
	if err := okxService.SyncTime(); err != nil {
		return fmt.Errorf("failed to sync time with OKX: %v", err)
	}

	// Create currency pair
	currencyPair := okxService.NewCurrencyPair(parts[0], parts[1])

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

	return nil
}
