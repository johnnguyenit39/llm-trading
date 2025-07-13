package transport

import (
	"j_ai_trade/common"
	"j_ai_trade/logger"
	"j_ai_trade/middlewares"
	"j_ai_trade/modules/auth/biz"
	dto "j_ai_trade/modules/auth/model/dto"
	"j_ai_trade/modules/auth/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SendForgotPasswordCode godoc
// @Summary Send reset password code
// @Description Send reset password code to user's email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param auth body dto.ForgotPasswordRequest true "Send reset password code"
// @Success 200 {object} common.BaseApiResponse[bool] "Reset password code sent successfully"
// @Failure 400 {object} common.BaseApiResponse[any] "Bad Request"
// @Failure 500 {object} common.BaseApiResponse[any] "Internal Server Error"
// @Router /v1/auth/send-forgot-password-code [post]
func SendForgotPasswordCode(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("ForgotPassword", c.GetString(middlewares.RequestIDKey))

		var input dto.ForgotPasswordRequest

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

		store := storage.NewPostgresStore(db)
		business := biz.NewRegisterBiz(store)

		err := business.SendResetPasswordCode(c.Request.Context(), &input)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to send reset password code")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[bool]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Reset password code sent successfully",
			Data:              true,
		})
		log.Info().Str("email", input.Email).Msg("Reset password code sent successfully")
	}
}
