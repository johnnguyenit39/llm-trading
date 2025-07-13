package ginApiKey

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/api_key/biz"
	"j_ai_trade/modules/api_key/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteApiKey godoc
// @Summary Delete new ApiKey
// @Description Delete a new ApiKey
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags ApiKey
// @Success 200 {object} model.ApiKey
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/delete/api-key/{id} [delete]
func DeleteApiKey(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeleteApiKeyBiz(store)
		data, err := business.DeleteApiKey(c.Request.Context(), id)

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
