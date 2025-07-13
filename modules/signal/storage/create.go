package storage

import (
	"context"
	"j_ai_trade/modules/signal/model"
)

func (postgresStore *postgresStore) CreateSignal(ctx context.Context, data *model.Signal) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
