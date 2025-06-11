package storage

import (
	"context"
	"j-ai-trade/modules/user/model"
)

func (postgresStore *postgresStore) GetUserByPhoneNumber(ctx context.Context, cond map[string]interface{}) (*model.User, error) {
	return nil, nil
}
