package ginOkx

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

// CancelOkxFuturesOrder godoc
// @Summary Cancel an OKX futures order
// @Description Cancel an existing futures order on OKX exchange by order ID
// @Accept json
// @Produce json
// @Tags Okx
// @Param request body dto.CancelFuturesOrderRequest true "OKX futures order cancellation parameters"
// @Success 200 {object} common.BaseApiResponse[string] "OKX futures order cancellation response"
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/okx/futures/order/cancel [post]
func CancelOkxFuturesOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("CancelOkxFuturesOrder", c.GetString(middlewares.RequestIDKey))

		var req dto.CancelFuturesOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid OKX futures order cancellation request: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind OKX futures order cancellation request")
			return
		}

		okxService := okx.GetInstance()
		business := biz.NewCancelFuturesOrderBiz(okxService)

		response, err := business.CancelFuturesOrder(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           "Failed to cancel OKX futures order: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to cancel OKX futures order")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[string]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "OKX futures order cancelled successfully",
			Data:              string(response),
		})
		log.Info().Msg("OKX futures order cancelled successfully")
	}
}
