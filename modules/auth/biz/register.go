package biz

import (
	"context"
	common "j-ai-trade/common"
	userModel "j-ai-trade/modules/user/model"
	"j-ai-trade/utils"

	"github.com/google/uuid"
)

type RegisterStorage interface {
	GetUserByEmail(ctx context.Context, cond map[string]interface{}) (*userModel.User, error)
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
	//FIXME: Get data from redis inthe future
	data.SubscriptionID = uuid.MustParse("4b60a017-0e68-4102-a9dc-b14f56d37294")
	data.Role = common.User
	data.Status = common.Active

	_, err = biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": data.PhoneNumber})

	if err == nil {
		return common.ErrorSimpleMessage("This email is already registered, please try again with other numbers.")
	}
	if err = biz.store.Register(ctx, data); err != nil {
		return err
	}
	return nil
}
