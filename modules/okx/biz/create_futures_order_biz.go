package biz

import (
	"context"
	"fmt"
	"j-ai-trade/brokers/okx"
	okxTypes "j-ai-trade/brokers/okx/types"
	dto "j-ai-trade/modules/okx/model/dto"
)

type CreateFuturesOrderBiz struct {
	okxService *okx.OKXService
}

func NewCreateFuturesOrderBiz(okxService *okx.OKXService) *CreateFuturesOrderBiz {
	return &CreateFuturesOrderBiz{
		okxService: okxService,
	}
}

func (biz *CreateFuturesOrderBiz) CreateFuturesOrder(ctx context.Context, req *dto.CreateFuturesOrderRequest) ([]byte, error) {
	// Create currency pair (always with USDT)
	pair := biz.okxService.NewCurrencyPair(req.Currency, "USDT")

	// Convert side string to OrderSide
	var side okxTypes.OrderSide
	switch req.Side {
	case "buy":
		side = okxTypes.Buy
	case "sell":
		side = okxTypes.Sell
	case "long":
		side = okxTypes.Long
	case "short":
		side = okxTypes.Short
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

	// Create the futures order with TP/SL values
	response, err := biz.okxService.CreateFuturesOrder(pair, req.Amount, req.Price, side, orderType, req.Leverage, req.PosSide, req.TpTriggerPx, req.TpOrdPx, req.SlTriggerPx, req.SlOrdPx)
	if err != nil {
		return nil, fmt.Errorf("failed to create futures order: %v", err)
	}

	return response, nil
}
