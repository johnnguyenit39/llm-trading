package storage

import (
	"context"
	"j-ai-trade/modules/order/model"
)

func (postgresStore *postgresStore) CreateOrder(ctx context.Context, data *model.Order) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
