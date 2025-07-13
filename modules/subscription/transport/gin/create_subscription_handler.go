package ginSubscription

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/subscription/biz"
	"j_ai_trade/modules/subscription/model"
	dto "j_ai_trade/modules/subscription/model/dto"
	"j_ai_trade/modules/subscription/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateSubscription godoc
// @Summary Create new Subscription
// @Description Create a new Subscription
// @Param Subscription body model.Subscription true "Create Subscription"
// @Produce application/json
// @Tags Subscription
// @Success 200 {object} model.Subscription
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/subscription/create [post]
func CreateSubscription(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.SubscriptionAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.Subscription{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateSubscriptionBiz(store)

		if err := business.CreateSubscription(c.Request.Context(), &data); err != nil {
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
			Message:           "Subscription created successfully",
			Data:              response,
		})
	}
}
