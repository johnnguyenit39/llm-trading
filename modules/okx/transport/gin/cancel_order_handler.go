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

// CancelOkxSpotOrder godoc
// @Summary Cancel an OKX spot order
// @Description Cancel an existing spot order on OKX exchange by order ID
// @Accept json
// @Produce json
// @Tags Okx
// @Param request body dto.CancelOrderRequest true "OKX spot order cancellation parameters"
// @Success 200 {object} common.BaseApiResponse[string] "OKX spot order cancellation response"
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/okx/spot/order/cancel [post]
func CancelOkxSpotOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("CancelOkxSpotOrder", c.GetString(middlewares.RequestIDKey))

		var req dto.CancelOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid OKX spot order cancellation request: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind OKX spot order cancellation request")
			return
		}

		okxService := okx.GetInstance()
		business := biz.NewCancelOrderBiz(okxService)

		response, err := business.CancelSpotOrder(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           "Failed to cancel OKX spot order: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to cancel OKX spot order")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[string]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "OKX spot order cancelled successfully",
			Data:              string(response),
		})
		log.Info().Msg("OKX spot order cancelled successfully")
	}
}
