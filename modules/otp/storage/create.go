package storage

import (
	"context"
	"j_ai_trade/modules/otp/model"
)

func (postgresStore *postgresStore) CreateOtp(ctx context.Context, data *model.Otp) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
