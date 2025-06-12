package ginOrder

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/order/biz"
	"j-ai-trade/modules/order/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteOrder godoc
// @Summary Delete new Order
// @Description Delete a new Order
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Order
// @Success 200 {object} model.Order
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/order/delete [delete]
func DeleteOrder(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeleteOrderBiz(store)
		data, err := business.DeleteOrder(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[any]{
			Success:           data,
			HttpRequestStatus: http.StatusOK,
			Message:           "Delete user successfully",
			Data:              nil,
		})
	}
}
