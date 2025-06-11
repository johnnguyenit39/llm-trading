package storage

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

func (postgresStore *postgresStore) GetMockById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error) {
	return nil, nil
}
