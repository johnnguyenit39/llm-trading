package biz

import (
	"context"
	"j-ai-trade/modules/otp/model"
)

type DeleteNewOtpStorage interface {
	GetOtpById(ctx context.Context, cond map[string]interface{}) (*model.Otp, error)
	DeleteOtp(ctx context.Context, cond map[string]interface{}) (bool, error)
	DeleteOtpByEmail(ctx context.Context, cond map[string]interface{}) (bool, error)
}

func NewDeleteOtpBiz(store DeleteNewOtpStorage) *deleteOtpBiz {
	return &deleteOtpBiz{store: store}
}

type deleteOtpBiz struct {
	store DeleteNewOtpStorage
}

func (biz *deleteOtpBiz) DeleteOtp(ctx context.Context, id string) (bool, error) {
	_, err := biz.store.GetOtpById(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	_, err = biz.store.DeleteOtp(ctx, map[string]interface{}{"id": id})

	if err != nil {
		return false, err
	}

	return true, nil
}

func (biz *deleteOtpBiz) DeleteOtpByEmail(ctx context.Context, email string) (bool, error) {
	_, err := biz.store.DeleteOtpByEmail(ctx, map[string]interface{}{"email": email})

	if err != nil {
		return false, err
	}

	return true, nil
}
