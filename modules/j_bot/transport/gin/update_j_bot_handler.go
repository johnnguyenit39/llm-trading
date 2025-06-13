package ginJbot

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/j_bot/biz"
	"j-ai-trade/modules/j_bot/model"
	dto "j-ai-trade/modules/j_bot/model/dto"
	"j-ai-trade/modules/j_bot/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateJbot godoc
// @Summary Update Jbot
// @Description Update Jbot
// @Param id path string true "User UUID" format(uuid)
// @Param user body dto.JbotUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags Jbot
// @Success 200 {object} dto.JbotGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/jbot/update/{id} [put]
func UpdateJbot(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input dto.JbotUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.Jbot
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdateJbotBiz(store)
		business.UpdateJbot(c.Request.Context(), id, &data)

		if err := business.UpdateJbot(c.Request.Context(), id, &data); err != nil {
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
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
