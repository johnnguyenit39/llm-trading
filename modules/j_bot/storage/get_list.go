package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/j_bot/model"
)

func (postgresStore *postgresStore) GetJbots(ctx context.Context, paging *common.Pagination) ([]model.Jbot, error) {
	var divisions []model.Jbot
	query := postgresStore.db.Model(&model.Jbot{})
	err := query.
		Where("deleted_at IS NULL").
		Count(&paging.Count).Error
	if err != nil {
		return nil, err
	}

	if (paging != &common.Pagination{}) {
		if paging.Size != 0 && paging.Index >= 0 {
			query = query.Offset(paging.Index).
				Limit(paging.Size)
		}
	}

	err = query.
		Where("deleted_at IS NULL").
		Find(&divisions).
		Error
	if err != nil {
		return nil, err
	}

	return divisions, nil
}
