package biz

import (
	"context"
	"j-ai-trade/modules/user/model"
	"j-ai-trade/utils"
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
	accessToken, err = utils.GeneSubscriptionJWT(userId)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = utils.GeneSubscriptionRefreshToken(userId)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
