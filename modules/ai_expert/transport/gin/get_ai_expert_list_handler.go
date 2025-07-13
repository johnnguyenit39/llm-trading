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

// GetAiExperts godoc
// @Summary Get a list of AiExpert
// @Description Retrieve a list of AiExperts based on provided filters and pagination
// @Produce json
// @Tags AiExpert
// @Param AiExpert body model.AiExpertGetListRequest true "Get AiExperts"  // Correctly specify the request body
// @Success 200 {array} model.AiExpert
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/ai-expert/list [post]
func GetAiExperts(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input model.AiExpertGetListRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		pagination := &common.Pagination{
			Size:  input.Pagination.Size,
			Index: input.Pagination.Index,
		}
		store := storage.NewPostgresStore(db)
		business := biz.NewGetAiExpertsBiz(store)
		list, err := business.GetAiExperts(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.AiExpertGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get AiExpert list successfully",
			Data: model.AiExpertGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
