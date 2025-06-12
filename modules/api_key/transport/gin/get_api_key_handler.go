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

// GetApiKey godoc
// @Summary Get ApiKey
// @Description Return ApiKey
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags ApiKey
// @Success 200 {object} model.ApiKey
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/api-key/get [get]
func GetApiKeyById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetApiKeyByIdBiz(store)
		_, err := business.GetApiKeyById(c.Request.Context(), id)

		if err != nil {
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
			Message:           "ApiKey user successfully",
			Data:              response,
		})
	}
}
