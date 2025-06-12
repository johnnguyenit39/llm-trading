package ginOrder

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/order/biz"
	dto "j-ai-trade/modules/order/model/dto"
	"j-ai-trade/modules/order/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetOrder godoc
// @Summary Get Order
// @Description Return Order
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Order
// @Success 200 {object} dto.OrderGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/order/get [get]
func GetOrderById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetOrderByIdBiz(store)
		_, err := business.GetOrderById(c.Request.Context(), id)

		if err != nil {
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
			Message:           "Order user successfully",
			Data:              response,
		})
	}
}
