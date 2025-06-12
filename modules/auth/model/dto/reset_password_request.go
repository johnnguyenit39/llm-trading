package dto

type ResetPasswordRequest struct {
	Email                   string `json:"email" binding:"required"`
	Code                    string `json:"code" binding:"required"`
	NewPassword             string `json:"new_password" binding:"required,min=8"`
	NewPasswordConfirmation string `json:"new_password_confirmation" binding:"required,min=8"`
}
