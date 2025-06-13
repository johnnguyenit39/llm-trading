package ginSubscription

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/subscription/biz"
	"j-ai-trade/modules/subscription/model"
	"j-ai-trade/modules/subscription/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetSubscription godoc
// @Summary Get Subscription
// @Description Return Subscription
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Subscription
// @Success 200 {object} model.Subscription
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/subscription/get/{id} [get]
func GetSubscriptionById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetSubscriptionByIdBiz(store)
		_, err := business.GetSubscriptionById(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := model.Subscription{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.Subscription]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Subscription user successfully",
			Data:              response,
		})
	}
}
