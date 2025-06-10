package model

type UserUpdatePasswordRequest struct {
	OldPassword             string `json:"old_password" binding:"required,min=8"`
	NewPassword             string `json:"new_password" binding:"required,min=8"`
	NewPasswordConfirmation string `json:"new_password_confirmation" binding:"required,min=8"`
}
