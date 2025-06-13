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

// UpdatePermission godoc
// @Summary Update Permission
// @Description Update Permission
// @Param id path string true "User UUID" format(uuid)
// @Param user body model.PermissionUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags Permission
// @Success 200 {object} model.Permission
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/permission/update/{id} [put]
func UpdatePermission(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input model.PermissionUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.Permission
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdatePermissionBiz(store)
		business.UpdatePermission(c.Request.Context(), id, &data)

		if err := business.UpdatePermission(c.Request.Context(), id, &data); err != nil {
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
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
