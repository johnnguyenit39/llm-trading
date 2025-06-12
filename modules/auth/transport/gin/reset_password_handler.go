package transport

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/auth/biz"
	dto "j-ai-trade/modules/auth/model/dto"
	"j-ai-trade/modules/auth/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ResetPassword godoc
// @Summary Reset password
// @Description Reset password with new password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param auth body dto.ResetPasswordRequest true "Reset password"
// @Success 200 {object} common.BaseApiResponse[bool] "Password reset successfully"
// @Failure 400 {object} common.BaseApiResponse[any] "Bad Request"
// @Failure 500 {object} common.BaseApiResponse[any] "Internal Server Error"
// @Router /v1/auth/reset-password [post]
func ResetPassword(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("ResetPassword", c.GetString(middlewares.RequestIDKey))

		var input dto.ResetPasswordRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind the request")
			return
		}

		if input.NewPassword != input.NewPasswordConfirmation {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "New password and confirmation do not match",
				Data:              nil,
			})
			return
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewRegisterBiz(store)

		err := business.ResetPassword(c.Request.Context(), &input)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to reset password")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[bool]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Password reset successfully",
			Data:              true,
		})
		log.Info().Str("email", input.Email).Msg("Password reset successfully")
	}
}
