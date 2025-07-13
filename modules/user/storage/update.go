package storage

import (
	"context"
	"j_ai_trade/modules/user/model"
)

func (postgresStore *postgresStore) UpdateUser(ctx context.Context, cond map[string]interface{}, dataUpdate *model.User) error {
	return nil
}
