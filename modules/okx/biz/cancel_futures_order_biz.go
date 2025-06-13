package biz

import (
	"context"
	"j-ai-trade/brokers/okx"
	dto "j-ai-trade/modules/okx/model/dto"
)

type CancelFuturesOrderBiz struct {
	okxService *okx.OKXService
}

func NewCancelFuturesOrderBiz(okxService *okx.OKXService) *CancelFuturesOrderBiz {
	return &CancelFuturesOrderBiz{
		okxService: okxService,
	}
}

func (biz *CancelFuturesOrderBiz) CancelFuturesOrder(ctx context.Context, req *dto.CancelFuturesOrderRequest) ([]byte, error) {
	// Create currency pair (always with USDT)
	pair := biz.okxService.NewCurrencyPair(req.Currency, "USDT")

	// Cancel the futures order
	response, err := biz.okxService.CancelFuturesOrder(req.OrderID, pair.Symbol)
	if err != nil {
		return nil, err
	}

	return response, nil
}
