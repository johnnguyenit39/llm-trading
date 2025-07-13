package biz

import (
	"context"
	"j_ai_trade/modules/ai_expert/model"
)

type AiExpertStorage interface {
	CreateAiExpert(ctx context.Context, data *model.AiExpert) error
}

func NewCreateAiExpertBiz(store AiExpertStorage) *createAiExpertBiz {
	return &createAiExpertBiz{store: store}
}

type createAiExpertBiz struct {
	store AiExpertStorage
}

func (biz *createAiExpertBiz) CreateAiExpert(ctx context.Context, data *model.AiExpert) error {
	if err := biz.store.CreateAiExpert(ctx, data); err != nil {
		return err
	}
	return nil
}
