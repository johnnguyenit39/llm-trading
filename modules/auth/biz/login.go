package biz

import (
	"context"
	common "j-okx-ai/common"
	dto "j-okx-ai/modules/auth/model/dto"
	userModel "j-okx-ai/modules/user/model"
	"j-okx-ai/utils"
)

type LoginStorage interface {
	GetUserByPhoneNumber(ctx context.Context, cond map[string]interface{}) (*userModel.User, error)
}

func NewLoginBiz(store LoginStorage) *createLoginBiz {
	return &createLoginBiz{store: store}
}

type createLoginBiz struct {
	store LoginStorage
}

func (biz *createLoginBiz) Login(ctx context.Context, input *userModel.User) (*dto.LoginResponse, error) {
	exsitedUser, err := biz.store.GetUserByPhoneNumber(ctx, map[string]interface{}{"phone_number": input.PhoneNumber})
	if err != nil {
		return nil, common.ErrEntityNotFoundEntity(userModel.EntityName, err)
	}

	if !utils.CheckPasswordHash(input.Password, exsitedUser.Password) {
		return nil, common.ErrorSimpleMessage("The password is incorrect please try again.")
	}

	token, err := utils.GeneMockJWT(exsitedUser.BaseModel.ID.Hex())
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GeneMockRefreshToken(exsitedUser.BaseModel.ID.Hex())
	if err != nil {
		return nil, err
	}
	return &dto.LoginResponse{
		User:         *exsitedUser,
		AccessToken:  token,
		RefreshToken: refreshToken,
	}, nil
}
