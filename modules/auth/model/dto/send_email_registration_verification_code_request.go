package dto

type SendEmailVerificationCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}
