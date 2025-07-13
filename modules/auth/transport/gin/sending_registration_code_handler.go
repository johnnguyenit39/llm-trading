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

// AuthLogin godoc
// @Summary Send registration code verification
// @Description Send registration code verification
// @Tags Authentication
// @Accept json
// @Produce json
// @Param auth body dto.SendEmailVerificationCodeRequest true "Send registration code verification"
// @Success 201 {object} common.BaseApiResponse[bool] "Sent registration code successfully"
// @Failure 400 {object} common.BaseApiResponse[any] "Bad Request"
// @Failure 500 {object} common.BaseApiResponse[any] "Internal Server Error"
// @Router /v1/auth/send-email-registration-code [post]
func SendEmailRegistrationCode(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("Send registration code", c.GetString(middlewares.RequestIDKey))

		var input dto.SendEmailVerificationCodeRequest

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

		data := dto.SendEmailVerificationCodeRequest{
			Email: input.Email,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewRegisterBiz(store)

		err := business.SendEmailRegistrationCode(c.Request.Context(), &data)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to send registration code")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[bool]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Sent registration code successfully",
			Data:              true,
		})
		log.Info().Str("email", input.Email).Msg("sent registration code successfully")

	}
}
