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

// AuthLogin godoc
// @Summary Verify registration code
// @Description Verify registration code
// @Tags Authentication
// @Accept json
// @Produce json
// @Param auth body dto.VerifyEmailRegistrationCodeRequest true "Verify registration code"
// @Success 201 {object} common.BaseApiResponse[bool] "Verified registration code successfully"
// @Failure 400 {object} common.BaseApiResponse[any] "Bad Request"
// @Failure 500 {object} common.BaseApiResponse[any] "Internal Server Error"
// @Router /v1/auth/verify-email-registration-code [post]
func EmailRegistrationCodeVerification(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("Login", c.GetString(middlewares.RequestIDKey))

		var input dto.VerifyEmailRegistrationCodeRequest

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

		data := dto.VerifyEmailRegistrationCodeRequest{
			Email: input.Email,
			Code:  input.Code,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewRegisterBiz(store)

		err := business.VerifyEmailCode(c.Request.Context(), &data)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed verify registration code")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[bool]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Verified registration code successfullyy",
			Data:              true,
		})
		log.Info().Str("otp", input.Code).Msg("Verified registration code successfully")

	}
}
