package ginPermission

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/permission/biz"
	"j-ai-trade/modules/permission/model"
	"j-ai-trade/modules/permission/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetPermission godoc
// @Summary Get Permission
// @Description Return Permission
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Permission
// @Success 200 {object} model.Permission
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/Permission/get [get]
func GetPermissionById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetPermissionByIdBiz(store)
		_, err := business.GetPermissionById(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := model.PermissionGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.PermissionGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Permission user successfully",
			Data:              response,
		})
	}
}
