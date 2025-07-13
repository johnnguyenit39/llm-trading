package storage

import (
	"context"
	"fmt"
	"j_ai_trade/brokers/okx"
	dto "j_ai_trade/modules/okx/model/dto"
)

func (postgresStore *postgresStore) CancelSpotOrder(ctx context.Context, req *dto.CancelOrderRequest) ([]byte, error) {
	// Get the OKX service instance
	okxService := okx.NewOKXService(nil)

	// Create currency pair (always with USDT)
	pair := okxService.NewCurrencyPair(req.Currency, "USDT")

	// Cancel the order
	response, err := okxService.CancelSpotOrder(req.OrderID, pair.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %v", err)
	}

	return response, nil
}
