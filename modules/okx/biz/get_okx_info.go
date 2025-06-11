package biz

import (
	"context"
	dto "j-ai-trade/modules/okx/model/dto"
)

type GetOkxInfoStorage interface {
	GetOkxInfo(ctx context.Context, cond map[string]interface{}) (*dto.OkxInfoResponse, error)
}

func NewGetOkxInfoBiz(store GetOkxInfoStorage) *getOkxInfoBiz {
	return &getOkxInfoBiz{store: store}
}

type getOkxInfoBiz struct {
	store GetOkxInfoStorage
}

func (biz *getOkxInfoBiz) GetOkxInfo(ctx context.Context) (*dto.OkxInfoResponse, error) {
	data, err := biz.store.GetOkxInfo(ctx, map[string]interface{}{})

	if err != nil {
		return nil, err
	}
	return data, nil
}
