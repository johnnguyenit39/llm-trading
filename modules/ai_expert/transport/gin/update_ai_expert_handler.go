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

// UpdateAiExpert godoc
// @Summary Update AiExpert
// @Description Update AiExpert
// @Param id path string true "User UUID" format(uuid)
// @Param user body model.AiExpertUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags AiExpert
// @Success 200 {object} model.AiExpert
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/ai-expert/update [put]
func UpdateAiExpert(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input model.AiExpertUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.AiExpert
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdateAiExpertBiz(store)
		business.UpdateAiExpert(c.Request.Context(), id, &data)

		if err := business.UpdateAiExpert(c.Request.Context(), id, &data); err != nil {
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
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
