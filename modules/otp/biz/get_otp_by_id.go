package biz

import (
	"context"
	"j_ai_trade/modules/otp/model"
)

type GetOtpByIdStorage interface {
	GetOtpById(ctx context.Context, cond map[string]interface{}) (*model.Otp, error)
}

func NewGetOtpByIdBiz(store GetOtpByIdStorage) *getOtpByIdBiz {
	return &getOtpByIdBiz{store: store}
}

type getOtpByIdBiz struct {
	store GetOtpByIdStorage
}

func (biz *getOtpByIdBiz) GetOtpById(ctx context.Context, id string) (*model.Otp, error) {
	data, err := biz.store.GetOtpById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return nil, err
	}
	return data, nil
}
