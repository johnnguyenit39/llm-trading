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

// CreateOkxFuturesOrder godoc
// @Summary Create a new OKX futures order
// @Description Create a new futures order on OKX exchange with specified parameters
// @Accept json
// @Produce json
// @Tags Okx
// @Param request body dto.CreateFuturesOrderRequest true "OKX futures order creation parameters"
// @Success 200 {object} common.BaseApiResponse[string] "OKX futures order creation response"
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/okx/futures/order/create [post]
func CreateOkxFuturesOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("CreateOkxFuturesOrder", c.GetString(middlewares.RequestIDKey))

		var req dto.CreateFuturesOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid OKX futures order request: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind OKX futures order request")
			return
		}

		okxService := okx.NewOKXService(nil)
		business := biz.NewCreateFuturesOrderBiz(okxService)

		response, err := business.CreateFuturesOrder(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           "Failed to create OKX futures order: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to create OKX futures order")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[string]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "OKX futures order created successfully",
			Data:              string(response),
		})
		log.Info().Msg("OKX futures order created successfully")
	}
}
