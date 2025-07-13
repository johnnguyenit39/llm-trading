package biz

import (
	"context"
	"j_ai_trade/modules/ai_expert/model"
)

type DeleteNewAiExpertStorage interface {
	GetAiExpertById(ctx context.Context, cond map[string]interface{}) (*model.AiExpert, error)
	DeleteAiExpert(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteAiExpertBiz(store DeleteNewAiExpertStorage) *deleteAiExpertBiz {
	return &deleteAiExpertBiz{store: store}
}

type deleteAiExpertBiz struct {
	store DeleteNewAiExpertStorage
}

func (biz *deleteAiExpertBiz) DeleteAiExpert(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetAiExpertById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteAiExpert(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}
