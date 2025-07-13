package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/api_key/model"
)

func (postgresStore *postgresStore) GetApiKeys(ctx context.Context, paging *common.Pagination) ([]model.ApiKey, error) {
	var divisions []model.ApiKey
	query := postgresStore.db.Model(&model.ApiKey{})
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
