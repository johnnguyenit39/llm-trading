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

// CreateOkxSpotOrder godoc
// @Summary Create a new OKX spot order
// @Description Create a new spot order on OKX exchange with specified parameters
// @Accept json
// @Produce json
// @Tags Okx
// @Param request body dto.CreateOrderRequest true "OKX spot order creation parameters"
// @Success 200 {object} common.BaseApiResponse[string] "OKX spot order creation response"
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/okx/spot/order/create [post]
func CreateOkxSpotOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("CreateOkxSpotOrder", c.GetString(middlewares.RequestIDKey))

		var req dto.CreateOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid OKX spot order request: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind OKX spot order request")
			return
		}

		okxService := okx.GetInstance()
		business := biz.NewCreateOrderBiz(okxService)

		response, err := business.CreateSpotOrder(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           "Failed to create OKX spot order: " + err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to create OKX spot order")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[string]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "OKX spot order created successfully",
			Data:              string(response),
		})
		log.Info().Msg("OKX spot order created successfully")
	}
}
