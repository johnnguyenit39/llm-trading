package biz

import (
	"context"
	"j-okx-ai/common"
	"j-okx-ai/modules/user/model"
	requestModel "j-okx-ai/modules/user/model/requests"
	"j-okx-ai/utils"
)

type UpdateUserPasswordStorage interface {
	GetUserById(ctx context.Context, cond map[string]interface{}) (*model.User, error)
	UpdateUserPassword(ctx context.Context, cond map[string]interface{}, dataUpdate *model.User) error
}

func NewUpdateUserPasswordBiz(store UpdateUserPasswordStorage) *updateUserPasswordBiz {
	return &updateUserPasswordBiz{store: store}
}

type updateUserPasswordBiz struct {
	store UpdateUserPasswordStorage
}

func (biz *updateUserPasswordBiz) UpdateUserPassword(ctx context.Context, id string, input *requestModel.UserUpdatePasswordRequest) (*model.User, error) {
	user, err := biz.store.GetUserById(ctx, map[string]interface{}{"_id": id})

	if err != nil {
		return nil, err
	}

	isMatched := utils.CheckPasswordHash(input.OldPassword, user.Password)
	if !isMatched {
		return nil, common.ErrorSimpleMessage("Current password is incorrect please try again.")
	}

	hashedNewPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		return nil, err
	}

	user.Password = hashedNewPassword

	if err := biz.store.UpdateUserPassword(ctx, map[string]interface{}{"_id": id}, user); err != nil {
		return nil, err
	}

	newData, err := biz.store.GetUserById(ctx, map[string]interface{}{"_id": id})
	if err != nil {
		return nil, err
	}

	return newData, nil
}
