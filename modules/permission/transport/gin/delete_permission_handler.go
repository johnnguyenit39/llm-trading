package ginPermission

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/permission/biz"
	"j-ai-trade/modules/permission/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeletePermission godoc
// @Summary Delete new Permission
// @Description Delete a new Permission
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Permission
// @Success 200 {object} model.Permission
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/Permission/delete [delete]
func DeletePermission(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeletePermissionBiz(store)
		data, err := business.DeletePermission(c.Request.Context(), id)

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
