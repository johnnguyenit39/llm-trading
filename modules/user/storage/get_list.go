package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/user/model"
)

func (postgresStore *postgresStore) GetUsers(ctx context.Context, paging *common.Pagination) ([]model.User, error) {
	return nil, nil
}
