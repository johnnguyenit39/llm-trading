package storage

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

func (postgresStore *postgresStore) CreateMock(ctx context.Context, data *model.Okx) error {
	return nil
}
