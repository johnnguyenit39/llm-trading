package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/user/model"
)

func (postgresStore *postgresStore) GetUsers(ctx context.Context, paging *common.Pagination) ([]model.User, error) {
	return nil, nil
}
