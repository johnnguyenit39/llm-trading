package storage

import (
	"context"
	"j_ai_trade/modules/subscription/model"
)

func (postgresStore *postgresStore) CreateSubscription(ctx context.Context, data *model.Subscription) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
