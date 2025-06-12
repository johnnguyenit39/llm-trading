package ginOrder

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/order/biz"
	dto "j-ai-trade/modules/order/model/dto"
	"j-ai-trade/modules/order/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetOrders godoc
// @Summary Get a list of Order
// @Description Retrieve a list of Orders based on provided filters and pagination
// @Produce json
// @Tags Order
// @Param Order body dto.OrderGetListRequest true "Get Orders"  // Correctly specify the request body
// @Success 200 {object} dto.OrderGetListResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/order/list [post]
func GetOrders(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.OrderGetListRequest

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
		business := biz.NewGetOrdersBiz(store)
		list, err := business.GetOrders(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.OrderGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Order list successfully",
			Data: dto.OrderGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
