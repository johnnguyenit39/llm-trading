package ginSubscription

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/subscription/biz"
	"j-ai-trade/modules/subscription/model"
	dto "j-ai-trade/modules/subscription/model/dto"
	"j-ai-trade/modules/subscription/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateSubscription godoc
// @Summary Update Subscription
// @Description Update Subscription
// @Param id path string true "User UUID" format(uuid)
// @Param user body dto.SubscriptionUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags Subscription
// @Success 200 {object} model.Subscription
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/subscription/update/{id} [put]
func UpdateSubscription(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input dto.SubscriptionUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.Subscription
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdateSubscriptionBiz(store)
		business.UpdateSubscription(c.Request.Context(), id, &data)

		if err := business.UpdateSubscription(c.Request.Context(), id, &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := dto.SubscriptionGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.SubscriptionGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
