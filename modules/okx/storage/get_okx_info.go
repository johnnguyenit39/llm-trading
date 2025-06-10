package storage

import (
	"context"
	"encoding/json"
	"fmt"
	dto "j-okx-ai/modules/okx/model/dto"
	"j-okx-ai/okx"
)

func (mongodbStore *mongodbStore) GetOkxInfo(ctx context.Context, cond map[string]interface{}) (*dto.OkxInfoResponse, error) {
	// Get the OKX service instance
	okxService := okx.GetInstance()

	// Get account information
	_, rawResponse, err := okxService.GetAccount("USDT")
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
