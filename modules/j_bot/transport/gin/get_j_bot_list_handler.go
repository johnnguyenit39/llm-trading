package ginJbot

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/j_bot/biz"
	dto "j_ai_trade/modules/j_bot/model/dto"
	"j_ai_trade/modules/j_bot/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetJbots godoc
// @Summary Get a list of Jbot
// @Description Retrieve a list of Jbots based on provided filters and pagination
// @Produce json
// @Tags Jbot
// @Param Jbot body dto.JbotGetListRequest true "Get Jbots"  // Correctly specify the request body
// @Success 200 {object} dto.JbotGetListResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/jbot/list [post]
func GetJbots(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.JbotGetListRequest

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
		business := biz.NewGetJbotsBiz(store)
		list, err := business.GetJbots(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.JbotGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Jbot list successfully",
			Data: dto.JbotGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
