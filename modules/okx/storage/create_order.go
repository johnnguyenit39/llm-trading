package storage

import (
	"context"
	"fmt"
	dto "j-ai-trade/modules/okx/model/dto"
	"j-ai-trade/okx"
)

func (mongodbStore *mongodbStore) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) ([]byte, error) {
	// Get the OKX service instance
	okxService := okx.GetInstance()

	// Create currency pair (always with USDT)
	pair := okxService.NewCurrencyPair(req.Currency, "USDT")

	// Convert side string to OrderSide
	var side okx.OrderSide
	switch req.Side {
	case "buy":
		side = okx.Buy
	case "sell":
		side = okx.Sell
	default:
		return nil, fmt.Errorf("invalid side: %s", req.Side)
	}

	// Convert type string to OrderType
	var orderType okx.OrderType
	switch req.Type {
	case "limit":
		orderType = okx.Limit
	case "market":
		orderType = okx.Market
	default:
		return nil, fmt.Errorf("invalid order type: %s", req.Type)
	}

	// Create the order
	response, err := okxService.CreateOrder(pair, req.Amount, req.Price, side, orderType)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	return response, nil
}
