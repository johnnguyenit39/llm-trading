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

// CreateOkxOrder godoc
// @Summary Create a new OKX order
// @Description Create a new order on OKX exchange with specified parameters
// @Accept json
// @Produce json
// @Tags Okx
// @Param request body dto.CreateOrderRequest true "OKX order creation parameters"
// @Success 200 {object} common.BaseApiResponse[string] "OKX order creation response"
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/okx/order/create [post]
func CreateOkxOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("CreateOkxOrder", c.GetString(middlewares.RequestIDKey))

		var req dto.CreateOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid OKX order request: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind OKX order request")
			return
		}

		okxService := okx.GetInstance()
		business := biz.NewCreateOrderBiz(okxService)

		response, err := business.CreateOrder(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           "Failed to create OKX order: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to create OKX order")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[string]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "OKX order created successfully",
			Data:              string(response),
		})
		log.Info().Msg("OKX order created successfully")
	}
}
