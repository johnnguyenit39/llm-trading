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

// GetPermissions godoc
// @Summary Get a list of Permission
// @Description Retrieve a list of Permissions based on provided filters and pagination
// @Produce json
// @Tags Permission
// @Param Permission body model.PermissionGetListRequest true "Get Permissions"  // Correctly specify the request body
// @Success 200 {array} model.Permission
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/Permission/list [post]
func GetPermissions(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input model.PermissionGetListRequest

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
		business := biz.NewGetPermissionsBiz(store)
		list, err := business.GetPermissions(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.PermissionGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Permission list successfully",
			Data: model.PermissionGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
