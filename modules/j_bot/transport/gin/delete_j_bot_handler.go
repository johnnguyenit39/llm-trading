package ginJbot

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/j_bot/biz"
	"j-ai-trade/modules/j_bot/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteJbot godoc
// @Summary Delete new Jbot
// @Description Delete a new Jbot
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Jbot
// @Success 200 {object} model.Jbot
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/jbot/delete [delete]
func DeleteJbot(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeleteJbotBiz(store)
		data, err := business.DeleteJbot(c.Request.Context(), id)

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
