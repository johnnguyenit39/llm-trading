package ginOrder

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/order/biz"
	"j_ai_trade/modules/order/model"
	dto "j_ai_trade/modules/order/model/dto"
	"j_ai_trade/modules/order/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateOrder godoc
// @Summary Update Order
// @Description Update Order
// @Param id path string true "User UUID" format(uuid)
// @Param user body dto.OrderUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags Order
// @Success 200 {object} dto.OrderGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/order/update/{id} [put]
func UpdateOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input dto.OrderUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.Order
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdateOrderBiz(store)
		business.UpdateOrder(c.Request.Context(), id, &data)

		if err := business.UpdateOrder(c.Request.Context(), id, &data); err != nil {
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
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
