package biz

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/ai_expert/model"
)

type GetAiExpertsStorage interface {
	GetAiExperts(ctx context.Context, paging *common.Pagination) ([]model.AiExpert, error)
}

func NewGetAiExpertsBiz(store GetAiExpertsStorage) *getAiExpertsBiz {
	return &getAiExpertsBiz{store: store}
}

type getAiExpertsBiz struct {
	store GetAiExpertsStorage
}

func (biz *getAiExpertsBiz) GetAiExperts(ctx context.Context, paging *common.Pagination) ([]model.AiExpert, error) {
	data, err := biz.store.GetAiExperts(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
