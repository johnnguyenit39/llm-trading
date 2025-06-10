package biz

import (
	"context"
	"j-okx-ai/modules/user/model"
	"j-okx-ai/utils"
)

type GetRefreshTokenStorage interface {
	RefreshToken(ctx context.Context, userId string) ([]model.User, error)
}

func NewRefreshTokenBiz() *getRefreshTokenBiz {
	return &getRefreshTokenBiz{}
}

type getRefreshTokenBiz struct {
}

func (biz *getRefreshTokenBiz) RefreshToken(userId string) (accessToken string, refreshToken string, err error) {
	accessToken, err = utils.GeneMockJWT(userId)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = utils.GeneMockRefreshToken(userId)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
