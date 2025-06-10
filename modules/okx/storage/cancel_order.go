package storage

import (
	"context"
	"fmt"
	dto "j-okx-ai/modules/okx/model/dto"
	"j-okx-ai/okx"
)

func (mongodbStore *mongodbStore) CancelOrder(ctx context.Context, req *dto.CancelOrderRequest) ([]byte, error) {
	// Get the OKX service instance
	okxService := okx.GetInstance()

	// Create currency pair (always with USDT)
	pair := okxService.NewCurrencyPair(req.Currency, "USDT")

	// Cancel the order
	response, err := okxService.CancelOrder(req.OrderID, pair.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %v", err)
	}

	return response, nil
}
