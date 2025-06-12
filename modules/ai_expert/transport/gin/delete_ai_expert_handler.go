package ginAiExpert

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/ai_expert/biz"
	"j-ai-trade/modules/ai_expert/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteAiExpert godoc
// @Summary Delete new AiExpert
// @Description Delete a new AiExpert
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags AiExpert
// @Success 200 {object} model.AiExpert
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/ai-expert/delete [delete]
func DeleteAiExpert(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeleteAiExpertBiz(store)
		data, err := business.DeleteAiExpert(c.Request.Context(), id)

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
