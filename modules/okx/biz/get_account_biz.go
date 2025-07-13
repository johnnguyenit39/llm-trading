package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"j_ai_trade/brokers/okx"
	dto "j_ai_trade/modules/okx/model/dto"
)

type GetAccountBiz struct {
	okxService *okx.OKXService
}

func NewGetAccountBiz(okxService *okx.OKXService) *GetAccountBiz {
	return &GetAccountBiz{
		okxService: okxService,
	}
}

func (biz *GetAccountBiz) GetAccount(ctx context.Context) (*dto.OkxInfoResponse, error) {
	// Get account information
	_, rawResponse, err := biz.okxService.GetAccount("USDT")
	if err != nil {
		return nil, err
	}

	// Parse the raw response into OkxInfoResponse
	var response dto.OkxInfoResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}
