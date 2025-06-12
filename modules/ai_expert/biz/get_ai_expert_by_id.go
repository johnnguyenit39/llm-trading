package biz

import (
	"context"
	"j-ai-trade/modules/ai_expert/model"
)

type GetAiExpertByIdStorage interface {
	GetAiExpertById(ctx context.Context, cond map[string]interface{}) (*model.AiExpert, error)
}

func NewGetAiExpertByIdBiz(store GetAiExpertByIdStorage) *getAiExpertByIdBiz {
	return &getAiExpertByIdBiz{store: store}
}

type getAiExpertByIdBiz struct {
	store GetAiExpertByIdStorage
}

func (biz *getAiExpertByIdBiz) GetAiExpertById(ctx context.Context, id string) (*model.AiExpert, error) {
	data, err := biz.store.GetAiExpertById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
