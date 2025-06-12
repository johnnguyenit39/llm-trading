package ginOrder

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/order/biz"
	"j-ai-trade/modules/order/model"
	dto "j-ai-trade/modules/order/model/dto"
	"j-ai-trade/modules/order/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateOrder godoc
// @Summary Create new Order
// @Description Create a new Order
// @Param Order body dto.OrderAddRequest true "Create Order"
// @Produce application/json
// @Tags Order
// @Success 200 {object} dto.OrderGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/order/create [post]
func CreateOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.OrderAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.Order{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateOrderBiz(store)

		if err := business.CreateOrder(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := dto.OrderGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.OrderGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Order created successfully",
			Data:              response,
		})
	}
}
