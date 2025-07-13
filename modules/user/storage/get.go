package storage

import (
	"context"
	"j_ai_trade/modules/user/model"
)

func (postgresStore *postgresStore) GetUserById(ctx context.Context, cond map[string]interface{}) (*model.User, error) {
	return nil, nil
}
