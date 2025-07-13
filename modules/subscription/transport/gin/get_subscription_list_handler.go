package ginSubscription

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/subscription/biz"
	dto "j_ai_trade/modules/subscription/model/dto"
	"j_ai_trade/modules/subscription/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetSubscriptions godoc
// @Summary Get a list of Subscription
// @Description Retrieve a list of Subscriptions based on provided filters and pagination
// @Produce json
// @Tags Subscription
// @Param Subscription body dto.SubscriptionGetListRequest true "Get Subscriptions"  // Correctly specify the request body
// @Success 200 {array} model.Subscription
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/subscription/list [post]
func GetSubscriptions(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.SubscriptionGetListRequest

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
		business := biz.NewGetSubscriptionsBiz(store)
		list, err := business.GetSubscriptions(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.SubscriptionGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Subscription list successfully",
			Data: dto.SubscriptionGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
