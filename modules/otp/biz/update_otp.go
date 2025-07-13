package biz

import (
	"context"
	"j_ai_trade/modules/otp/model"
)

type UpdateOtpStorage interface {
	GetOtpById(ctx context.Context, cond map[string]interface{}) (*model.Otp, error)
	UpdateOtp(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Otp) error
}

func NewUpdateOtpBiz(store UpdateOtpStorage) *updateOtpBiz {
	return &updateOtpBiz{store: store}
}

type updateOtpBiz struct {
	store UpdateOtpStorage
}

func (biz *updateOtpBiz) UpdateOtp(ctx context.Context, id string, dataUpdate *model.Otp) error {
	_, err := biz.store.GetOtpById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return err
	}

	if err := biz.store.UpdateOtp(ctx, map[string]interface{}{"id": id}, dataUpdate); err != nil {
		return err
	}

	return nil
}
