package biz

import (
	"context"
	common "j-okx-ai/common"
	userModel "j-okx-ai/modules/user/model"
	"j-okx-ai/utils"
)

type RegisterStorage interface {
	GetUserByPhoneNumber(ctx context.Context, cond map[string]interface{}) (*userModel.User, error)
	Register(ctx context.Context, data *userModel.User) error
}

func NewRegisterBiz(store RegisterStorage) *createRegisterBiz {
	return &createRegisterBiz{store: store}
}

type createRegisterBiz struct {
	store RegisterStorage
}

func (biz *createRegisterBiz) Register(ctx context.Context, data *userModel.User) error {
	hashedPassword, err := utils.HashPassword(data.Password)
	if err != nil {
		return err
	}
	data.Password = hashedPassword

	_, err = biz.store.GetUserByPhoneNumber(ctx, map[string]interface{}{"phone_number": data.PhoneNumber})

	if err == nil {
		return common.ErrorSimpleMessage("This phone number is already registered, please try again with other numbers.")
	}
	if err = biz.store.Register(ctx, data); err != nil {
		return err
	}
	return nil
}
