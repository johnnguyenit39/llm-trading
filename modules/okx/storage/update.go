package storage

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

func (postgresStore *postgresStore) UpdateSubscription(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Okx) error {
	return nil
}
