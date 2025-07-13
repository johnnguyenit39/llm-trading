package biz

import (
	"context"
	"j_ai_trade/modules/ai_expert/model"
)

type UpdateAiExpertStorage interface {
	GetAiExpertById(ctx context.Context, cond map[string]interface{}) (*model.AiExpert, error)
	UpdateAiExpert(ctx context.Context, cond map[string]interface{}, dataUpdate *model.AiExpert) error
}

func NewUpdateAiExpertBiz(store UpdateAiExpertStorage) *updateAiExpertBiz {
	return &updateAiExpertBiz{store: store}
}

type updateAiExpertBiz struct {
	store UpdateAiExpertStorage
}

func (biz *updateAiExpertBiz) UpdateAiExpert(ctx context.Context, id string, dataUpdate *model.AiExpert) error {
	_, err := biz.store.GetAiExpertById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateAiExpert(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
