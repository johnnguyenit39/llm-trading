package storage

import (
	"context"
	"j-ai-trade/modules/okx/model"
)

func (postgresStore *postgresStore) UpdateMock(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Okx) error {
	return nil
}
