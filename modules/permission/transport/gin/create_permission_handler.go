package ginPermission

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/permission/biz"
	"j_ai_trade/modules/permission/model"
	"j_ai_trade/modules/permission/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreatePermission godoc
// @Summary Create new Permission
// @Description Create a new Permission
// @Param Permission body model.Permission true "Create Permission"
// @Produce application/json
// @Tags Permission
// @Success 200 {object} model.Permission
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/permission/create [post]
func CreatePermission(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input model.PermissionAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.Permission{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreatePermissionBiz(store)

		if err := business.CreatePermission(c.Request.Context(), &data); err != nil {
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
			Message:           "Permission created successfully",
			Data:              response,
		})
	}
}
