package storage

import (
	"context"
	"j-ai-trade/modules/user/model"
)

func (postgresStore *postgresStore) UpdateUserPassword(ctx context.Context, cond map[string]interface{}, dataUpdate *model.User) error {
	return nil
}
