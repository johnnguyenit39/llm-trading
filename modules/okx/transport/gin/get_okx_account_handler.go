package ginOkx

import (
	"j_ai_trade/brokers/okx"
	"j_ai_trade/common"
	"j_ai_trade/logger"
	"j_ai_trade/middlewares"
	"j_ai_trade/modules/okx/biz"
	dto "j_ai_trade/modules/okx/model/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetOkxAccount godoc
// @Summary Get OKX account information
// @Description Retrieve OKX account balance and details including available funds and positions
// @Produce json
// @Tags Okx
// @Success 200 {object} common.BaseApiResponse[dto.OkxInfoResponse] "OKX account information"
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/okx/account/get [get]
func GetOkxAccount(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("GetOkxAccount", c.GetString(middlewares.RequestIDKey))

		okxService := okx.NewOKXService(nil)
		business := biz.NewGetAccountBiz(okxService)

		response, err := business.GetAccount(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           "Failed to get OKX account information: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to get OKX account information")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[dto.OkxInfoResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "OKX account information retrieved successfully",
			Data:              *response,
		})
		log.Info().Msg("OKX account information retrieved successfully")
	}
}
