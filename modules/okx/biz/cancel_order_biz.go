package biz

import (
	"context"
	"j-ai-trade/brokers/okx"
	dto "j-ai-trade/modules/okx/model/dto"
)

type CancelOrderBiz struct {
	okxService *okx.OKXService
}

func NewCancelOrderBiz(okxService *okx.OKXService) *CancelOrderBiz {
	return &CancelOrderBiz{
		okxService: okxService,
	}
}

func (biz *CancelOrderBiz) CancelOrder(ctx context.Context, req *dto.CancelOrderRequest) ([]byte, error) {
	// Create currency pair (always with USDT)
	pair := biz.okxService.NewCurrencyPair(req.Currency, "USDT")

	// Cancel the order
	response, err := biz.okxService.CancelOrder(req.OrderID, pair.Symbol)
	if err != nil {
		return nil, err
	}

	return response, nil
}
