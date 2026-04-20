package biz

import (
	"context"
	"j_ai_trade/modules/user/model"
	"j_ai_trade/utils"
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
	accessToken, err = utils.GenerateAccessToken(userId)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = utils.GenerateRefreshToken(userId)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
