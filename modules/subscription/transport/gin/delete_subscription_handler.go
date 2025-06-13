package ginSubscription

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/subscription/biz"
	"j-ai-trade/modules/subscription/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteSubscription godoc
// @Summary Delete new Subscription
// @Description Delete a new Subscription
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Subscription
// @Success 200 {object} model.Subscription
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/subscription/delete/{id} [delete]
func DeleteSubscription(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeleteSubscriptionBiz(store)
		data, err := business.DeleteSubscription(c.Request.Context(), id)

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
