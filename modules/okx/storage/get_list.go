package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/okx/model"
)

func (postgresStore *postgresStore) GetMocks(ctx context.Context, paging *common.Pagination) ([]model.Okx, error) {
	return nil, nil
}
