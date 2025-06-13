package ginAiExpert

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/ai_expert/biz"
	"j-ai-trade/modules/ai_expert/model"
	"j-ai-trade/modules/ai_expert/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAiExpert godoc
// @Summary Get AiExpert
// @Description Return AiExpert
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags AiExpert
// @Success 200 {object} model.AiExpert
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/ai-expert/get/{id} [get]
func GetAiExpertById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetAiExpertByIdBiz(store)
		_, err := business.GetAiExpertById(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := model.AiExpertGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.AiExpertGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "AiExpert user successfully",
			Data:              response,
		})
	}
}
