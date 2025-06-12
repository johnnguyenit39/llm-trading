package storage

import (
	"context"
	"j-ai-trade/common"
	"j-ai-trade/modules/auth/model/dto"
	otpModel "j-ai-trade/modules/otp/model"
	"j-ai-trade/modules/user/model"
	userModel "j-ai-trade/modules/user/model"
	"time"

	"gorm.io/gorm"
)

func (postgresStore *postgresStore) Register(ctx context.Context, data *model.User) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return common.ErrDB(err)
	}
	return nil
}

func (postgresStore *postgresStore) SendEmailRegistrationCode(ctx context.Context, data *otpModel.Otp) error {
	if err := postgresStore.db.Create(data).Error; err != nil {
		return common.ErrDB(err)
	}
	return nil
}

func (postgresStore *postgresStore) VerifyEmailRegistrationCode(ctx context.Context, userID string, data *dto.VerifyEmailRegistrationCodeRequest) error {
	//FIXME: Move more logic to biz layer
	otp := otpModel.Otp{}
	err := postgresStore.db.First(&otp, "user_id = ? AND code = ? AND type = ?", userID, data.Code, common.RegistrationOTP).Error

	if err != nil {
		return common.ErrDB(err)
	}

	if otp.ExpiresAt.Before(time.Now().UTC()) {
		return common.ErrorSimpleMessage("The code has expired, please try again.")
	}

	return nil
}

func (postgresStore *postgresStore) DeleteOtpByUserID(ctx context.Context, cond map[string]interface{}) (bool, error) {
	var data otpModel.Otp
	if err := postgresStore.db.Where(cond).Delete(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return false, err
	}
	return true, nil
}

func (postgresStore *postgresStore) UpdateUserEmailVerificationStatus(ctx context.Context, cond map[string]interface{}, dataUpdate *model.User) error {
	var user userModel.User
	return postgresStore.db.Model(&user).Where(cond).Updates(dataUpdate).Error
}
