package storage

import (
	"context"
	"j-ai-trade/modules/ai_expert/model"
)

func (postgresStore *postgresStore) CreateAiExpert(ctx context.Context, data *model.AiExpert) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return err
	}
	return nil
}
