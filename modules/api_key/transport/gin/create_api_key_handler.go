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

// CreateApiKey godoc
// @Summary Create new ApiKey
// @Description Create a new ApiKey
// @Param ApiKey body model.ApiKey true "Create ApiKey"
// @Produce application/json
// @Tags ApiKey
// @Success 200 {object} model.ApiKey
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/api-key/create [post]
func CreateApiKey(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input model.ApiKeyAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.ApiKey{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateApiKeyBiz(store)

		if err := business.CreateApiKey(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := model.ApiKeyGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.ApiKeyGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "ApiKey created successfully",
			Data:              response,
		})
	}
}
