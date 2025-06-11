package ginApiKey

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/api_key/biz"
	"j-ai-trade/modules/api_key/model"
	"j-ai-trade/modules/api_key/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetApiKeys godoc
// @Summary Get a list of ApiKey
// @Description Retrieve a list of ApiKeys based on provided filters and pagination
// @Produce json
// @Tags ApiKey
// @Param ApiKey body model.ApiKeyGetListRequest true "Get ApiKeys"  // Correctly specify the request body
// @Success 200 {array} model.ApiKey
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/api-key/list [post]
func GetApiKeys(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input model.ApiKeyGetListRequest

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
		business := biz.NewGetApiKeysBiz(store)
		list, err := business.GetApiKeys(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.ApiKeyGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get ApiKey list successfully",
			Data: model.ApiKeyGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
