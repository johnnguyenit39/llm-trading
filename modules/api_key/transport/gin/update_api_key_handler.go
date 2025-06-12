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

// UpdateApiKey godoc
// @Summary Update ApiKey
// @Description Update ApiKey
// @Param id path string true "User UUID" format(uuid)
// @Param user body model.ApiKeyUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags ApiKey
// @Success 200 {object} model.ApiKey
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/api-key/update [put]
func UpdateApiKey(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input model.ApiKeyUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.ApiKey
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdateApiKeyBiz(store)
		business.UpdateApiKey(c.Request.Context(), id, &data)

		if err := business.UpdateApiKey(c.Request.Context(), id, &data); err != nil {
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
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
