package biz

import (
	"context"
	common "j_ai_trade/common"
	dto "j_ai_trade/modules/auth/model/dto"
	userModel "j_ai_trade/modules/user/model"
	"j_ai_trade/utils"
)

type LoginStorage interface {
	GetUserByEmail(ctx context.Context, cond map[string]interface{}) (*userModel.User, error)
}

func NewLoginBiz(store LoginStorage) *createLoginBiz {
	return &createLoginBiz{store: store}
}

type createLoginBiz struct {
	store LoginStorage
}

func (biz *createLoginBiz) Login(ctx context.Context, input *userModel.User) (*dto.LoginResponse, error) {
	exsitedUser, err := biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": input.Email})
	if err != nil {
		return nil, common.ErrEntityNotFoundEntity(userModel.EntityName, err)
	}

	if !utils.CheckPasswordHash(input.Password, exsitedUser.Password) {
		return nil, common.ErrorSimpleMessage("The password is incorrect please try again.")
	}

	token, err := utils.GenerateAccessToken(exsitedUser.BaseModel.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(exsitedUser.BaseModel.ID.String())
	if err != nil {
		return nil, err
	}
	return &dto.LoginResponse{
		User:         *exsitedUser,
		AccessToken:  token,
		RefreshToken: refreshToken,
	}, nil
}
