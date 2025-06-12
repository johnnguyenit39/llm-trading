package biz

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/otp/model"
)

type GetOtpsStorage interface {
	GetOtps(ctx context.Context, paging *common.Pagination) ([]model.Otp, error)
}

func NewGetOtpsBiz(store GetOtpsStorage) *getOtpsBiz {
	return &getOtpsBiz{store: store}
}

type getOtpsBiz struct {
	store GetOtpsStorage
}

func (biz *getOtpsBiz) GetOtps(ctx context.Context, paging *common.Pagination) ([]model.Otp, error) {
	data, err := biz.store.GetOtps(ctx, paging)

	if err != nil {
		return nil, err
	}
	return data, nil
}
