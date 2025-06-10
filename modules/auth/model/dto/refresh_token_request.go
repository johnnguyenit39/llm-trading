package dto

type RefreshTokenRequest struct {
	Token        string `json:"token" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required,min=8"`
}
