package biz

import (
	"context"
	"fmt"
	common "j-ai-trade/common"
	emailHelper "j-ai-trade/helpers/email"
	dto "j-ai-trade/modules/auth/model/dto"
	otpModel "j-ai-trade/modules/otp/model"
	userModel "j-ai-trade/modules/user/model"
	"j-ai-trade/utils"
	"log"
	"time"

	"github.com/google/uuid"
)

type RegisterStorage interface {
	GetUserByEmail(ctx context.Context, cond map[string]interface{}) (*userModel.User, error)
	Register(ctx context.Context, data *userModel.User) error
	SendEmailRegistrationCode(ctx context.Context, data *otpModel.Otp) error
	VerifyEmailCode(ctx context.Context, userID string, otpType string, data *dto.VerifyEmailRegistrationCodeRequest) error
	DeleteOtpByUserID(ctx context.Context, cond map[string]interface{}) (bool, error)
	UpdateUserEmailVerificationStatus(ctx context.Context, cond map[string]interface{}, dataUpdate *userModel.User) error
	UpdateUserPassword(ctx context.Context, cond map[string]interface{}, dataUpdate *userModel.User) error
}

func NewRegisterBiz(store RegisterStorage) *createRegisterBiz {
	return &createRegisterBiz{store: store}
}

type createRegisterBiz struct {
	store RegisterStorage
}

func (biz *createRegisterBiz) Register(ctx context.Context, data *userModel.User) error {
	hashedPassword, err := utils.HashPassword(data.Password)
	if err != nil {
		return err
	}
	data.Password = hashedPassword
	//FIXME: Get data from redis inthe future
	data.SubscriptionID = uuid.MustParse("4b60a017-0e68-4102-a9dc-b14f56d37294")
	data.Role = string(common.User)
	data.Status = string(common.Active)

	_, err = biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": data.PhoneNumber})

	if err == nil {
		return common.ErrorSimpleMessage("This email is already registered, please try again with other numbers.")
	}
	if err = biz.store.Register(ctx, data); err != nil {
		return err
	}
	return nil
}

func (biz *createRegisterBiz) SendEmailRegistrationCode(ctx context.Context, data *dto.SendEmailVerificationCodeRequest) error {
	user, err := biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": data.Email})

	if err != nil {
		return common.ErrorSimpleMessage("The email is not registered, please try again with other email.")
	}

	err = biz.DeleteOtpByUserID(ctx, user.ID.String())

	if err != nil {
		return common.ErrorSimpleMessage("Something went wrong, please try again later.")
	}

	// Save otp to db
	now := time.Now().UTC()

	// Create new otp
	newOtp := otpModel.Otp{
		UserID:    user.ID,
		Code:      common.GenerateRandomCode(),
		ExpiresAt: now.Add(common.OTPExpiredTime),
		Used:      false,
		Type:      string(common.RegistrationOTP),
		BaseModel: common.BaseModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// Send otp to user
	emailContent := fmt.Sprintf(
		"Hello,\n\n"+
			"You recently requested to register an account. Please use the OTP below to complete your request:\n\n"+
			"Your OTP: %s\n\n"+
			"Please do not share this code with anyone. For security purposes, this OTP will expire in 10 minutes.\n\n"+
			"Thank you,\n"+
			"J-AI-Trade Support Team", newOtp.Code)

	if err := emailHelper.SendCustomEmail([]string{user.Email}, "Register OTP", emailContent, nil); err != nil {
		log.Printf("Error sending otp to user: %v", err)
		return err
	}

	if err := biz.store.SendEmailRegistrationCode(ctx, &newOtp); err != nil {
		return err
	}

	return nil
}

func (biz *createRegisterBiz) VerifyEmailCode(ctx context.Context, data *dto.VerifyEmailRegistrationCodeRequest) error {
	user, err := biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": data.Email})

	if err != nil {
		return common.ErrorSimpleMessage("The email is not registered, please try again with other email.")
	}

	if err := biz.store.VerifyEmailCode(ctx, user.ID.String(), string(common.RegistrationOTP), data); err != nil {
		return err
	}

	err = biz.DeleteOtpByUserID(ctx, user.ID.String())

	if err != nil {
		return common.ErrorSimpleMessage("Something went wrong, please try again later.")
	}

	if err := biz.UpdateVerifyUserEmail(ctx, data.Email); err != nil {
		return err
	}

	return nil
}

func (biz *createRegisterBiz) DeleteOtpByUserID(ctx context.Context, userID string) error {
	_, err := biz.store.DeleteOtpByUserID(ctx, map[string]interface{}{"user_id": userID})

	if err != nil {
		return err
	}

	return nil
}

func (biz *createRegisterBiz) UpdateVerifyUserEmail(ctx context.Context, email string) error {
	user, err := biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": email})
	user.IsEmailVerified = true
	if err != nil {
		return common.ErrorSimpleMessage("The email is not registered, please try again with other email.")
	}

	if err := biz.store.UpdateUserEmailVerificationStatus(ctx, map[string]interface{}{"id": user.ID}, user); err != nil {
		return err
	}

	return nil
}

func (biz *createRegisterBiz) SendResetPasswordCode(ctx context.Context, data *dto.ForgotPasswordRequest) error {
	user, err := biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": data.Email})

	if err != nil {
		return common.ErrorSimpleMessage("The email is not registered, please try again with other email.")
	}

	err = biz.DeleteOtpByUserID(ctx, user.ID.String())

	if err != nil {
		return common.ErrorSimpleMessage("Something went wrong, please try again later.")
	}

	// Save otp to db
	now := time.Now().UTC()

	// Create new otp
	newOtp := otpModel.Otp{
		UserID:    user.ID,
		Code:      common.GenerateRandomCode(),
		ExpiresAt: now.Add(common.OTPExpiredTime),
		Used:      false,
		Type:      string(common.ResetPasswordOTP),
		BaseModel: common.BaseModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// Send otp to user
	emailContent := fmt.Sprintf(
		"Hello,\n\n"+
			"You recently requested to reset your password. Please use the OTP below to complete your request:\n\n"+
			"Your OTP: %s\n\n"+
			"Please do not share this code with anyone. For security purposes, this OTP will expire in 10 minutes.\n\n"+
			"Thank you,\n"+
			"J-AI-Trade Support Team", newOtp.Code)

	if err := emailHelper.SendCustomEmail([]string{user.Email}, "Reset Password OTP", emailContent, nil); err != nil {
		log.Printf("Error sending otp to user: %v", err)
		return err
	}

	if err := biz.store.SendEmailRegistrationCode(ctx, &newOtp); err != nil {
		return err
	}

	return nil
}

func (biz *createRegisterBiz) VerifyResetPasswordCode(ctx context.Context, data *dto.VerifyResetPasswordCodeRequest) error {
	user, err := biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": data.Email})

	if err != nil {
		return common.ErrorSimpleMessage("The email is not registered, please try again with other email.")
	}

	err = biz.store.VerifyEmailCode(ctx, user.ID.String(), string(common.ResetPasswordOTP), &dto.VerifyEmailRegistrationCodeRequest{
		Email: data.Email,
		Code:  data.Code,
	})

	if err != nil {
		return err
	}

	return nil
}

func (biz *createRegisterBiz) ResetPassword(ctx context.Context, data *dto.ResetPasswordRequest) error {
	user, err := biz.store.GetUserByEmail(ctx, map[string]interface{}{"email": data.Email})

	if err != nil {
		return common.ErrorSimpleMessage("The email is not registered, please try again with other email.")
	}

	// Verify the OTP first
	err = biz.store.VerifyEmailCode(ctx, user.ID.String(), string(common.ResetPasswordOTP), &dto.VerifyEmailRegistrationCodeRequest{
		Email: data.Email,
		Code:  data.Code,
	})

	if err != nil {
		return err
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(data.NewPassword)
	if err != nil {
		return err
	}

	// Update the password
	user.Password = hashedPassword
	if err := biz.store.UpdateUserPassword(ctx, map[string]interface{}{"id": user.ID}, user); err != nil {
		return err
	}

	// Delete the used OTP
	err = biz.DeleteOtpByUserID(ctx, user.ID.String())
	if err != nil {
		return common.ErrorSimpleMessage("Something went wrong, please try again later.")
	}

	return nil
}
