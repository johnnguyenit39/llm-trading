package ginAiExpert

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/ai_expert/biz"
	"j_ai_trade/modules/ai_expert/model"
	"j_ai_trade/modules/ai_expert/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAiExpert godoc
// @Summary Create new AiExpert
// @Description Create a new AiExpert
// @Param AiExpert body model.AiExpert true "Create AiExpert"
// @Produce application/json
// @Tags AiExpert
// @Success 200 {object} model.AiExpert
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/ai-expert/create [post]
func CreateAiExpert(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input model.AiExpertAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.AiExpert{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateAiExpertBiz(store)

		if err := business.CreateAiExpert(c.Request.Context(), &data); err != nil {
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
			Message:           "AiExpert created successfully",
			Data:              response,
		})
	}
}
