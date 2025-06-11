package dto

import (
	userModel "j-ai-trade/modules/user/model"
)

type LoginResponse struct {
	User         userModel.User `json:"user"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
}
