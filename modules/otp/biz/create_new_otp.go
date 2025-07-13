package biz

import (
	"context"
	"j_ai_trade/modules/otp/model"
)

type OtpStorage interface {
	CreateOtp(ctx context.Context, data *model.Otp) error
}

func NewCreateOtpBiz(store OtpStorage) *createOtpBiz {
	return &createOtpBiz{store: store}
}

type createOtpBiz struct {
	store OtpStorage
}

func (biz *createOtpBiz) CreateOtp(ctx context.Context, data *model.Otp) error {
	if err := biz.store.CreateOtp(ctx, data); err != nil {
		return err
	}
	return nil
}
