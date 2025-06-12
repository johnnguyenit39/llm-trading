package dto

type VerifyResetPasswordCodeRequest struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}
