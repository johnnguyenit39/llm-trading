package storage

import (
	"context"
	"fmt"
	"j-ai-trade/brokers/okx"
	okxTypes "j-ai-trade/brokers/okx/types"
	dto "j-ai-trade/modules/okx/model/dto"
)

func (postgresStore *postgresStore) CreateSpotOrder(ctx context.Context, req *dto.CreateOrderRequest) ([]byte, error) {
	// Get the OKX service instance
	okxService := okx.NewOKXService(nil)

	// Create currency pair (always with USDT)
	pair := okxService.NewCurrencyPair(req.Currency, "USDT")

	// Convert side string to OrderSide
	var side okxTypes.OrderSide
	switch req.Side {
	case "buy":
		side = okxTypes.Buy
	case "sell":
		side = okxTypes.Sell
	default:
		return nil, fmt.Errorf("invalid side: %s", req.Side)
	}

	// Convert type string to OrderType
	var orderType okxTypes.OrderType
	switch req.Type {
	case "limit":
		orderType = okxTypes.Limit
	case "market":
		orderType = okxTypes.Market
	default:
		return nil, fmt.Errorf("invalid order type: %s", req.Type)
	}

	// Create the order
	response, err := okxService.CreateSpotOrder(pair, req.Amount, req.Price, side, orderType)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	return response, nil
}
