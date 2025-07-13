package ginSignal

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/signal/biz"
	"j_ai_trade/modules/signal/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteSignal godoc
// @Summary Delete new Signal
// @Description Delete a new Signal
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Signal
// @Success 200 {object} model.Signal
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/signal/delete/{id} [delete]
func DeleteSignal(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeleteSignalBiz(store)
		data, err := business.DeleteSignal(c.Request.Context(), id)

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
