package ginNovel

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/okx/biz"
	model "j-ai-trade/modules/okx/model"
	"j-ai-trade/modules/okx/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetSubscription godoc
// @Summary Get Okx
// @Description Return Okx
// @Param id path string true "Okx ID" // Updated to just type string
// @Produce application/json
// @Tags Okx
// @Success 200 {object} model.Okx
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/get/okx/{id} [get]
func GetSubscriptionById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("GetSubscriptionById", c.GetString(middlewares.RequestIDKey))

		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "id is required",
				Data:              nil,
			})
			log.Error().Msg("id is required")
			return
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewGetSubscriptionByIdBiz(store)

		data, err := business.GetSubscriptionById(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Str("Subscription_id", id).Msg("failed to get Okx by id")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[model.Okx]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Okx user successfully",
			Data:              *data,
		})
		log.Info().Str("Subscription_id", id).Msg("get Okx by id successfully")
	}
}
