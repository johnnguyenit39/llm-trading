package storage

import (
	"context"
	"j_ai_trade/common"
	"j_ai_trade/modules/subscription/model"
)

func (postgresStore *postgresStore) GetSubscriptions(ctx context.Context, paging *common.Pagination) ([]model.Subscription, error) {
	var divisions []model.Subscription
	query := postgresStore.db.Model(&model.Subscription{})
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
