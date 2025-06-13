package ginJbot

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/j_bot/biz"
	dto "j-ai-trade/modules/j_bot/model/dto"
	"j-ai-trade/modules/j_bot/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetJbot godoc
// @Summary Get Jbot
// @Description Return Jbot
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Jbot
// @Success 200 {object} dto.JbotGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/jbot/get/{id} [get]
func GetJbotById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetJbotByIdBiz(store)
		_, err := business.GetJbotById(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := dto.JbotGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.JbotGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Jbot user successfully",
			Data:              response,
		})
	}
}
