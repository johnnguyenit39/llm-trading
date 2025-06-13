package ginNovel

import (
	"j-ai-trade/brokers/okx"
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/okx/biz"
	dto "j-ai-trade/modules/okx/model/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CancelOkxOrder godoc
// @Summary Cancel an OKX order
// @Description Cancel an existing order on OKX exchange by order ID
// @Accept json
// @Produce json
// @Tags Okx
// @Param request body dto.CancelOrderRequest true "OKX order cancellation parameters"
// @Success 200 {object} common.BaseApiResponse[string] "OKX order cancellation response"
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/okx/order/cancel [post]
func CancelOkxOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("CancelOkxOrder", c.GetString(middlewares.RequestIDKey))

		var req dto.CancelOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid OKX order cancellation request: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind OKX order cancellation request")
			return
		}

		okxService := okx.GetInstance()
		business := biz.NewCancelOrderBiz(okxService)

		response, err := business.CancelSpotOrder(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           "Failed to cancel OKX order: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to cancel OKX order")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[string]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "OKX order cancelled successfully",
			Data:              string(response),
		})
		log.Info().Msg("OKX order cancelled successfully")
	}
}
