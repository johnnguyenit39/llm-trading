package ginNovel

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/okx/biz"
	model "j-ai-trade/modules/okx/model"
	dto "j-ai-trade/modules/okx/model/dto"
	"j-ai-trade/modules/okx/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateSubscription godoc
// @Summary Create new Okx
// @Description Create a new Okx
// @Param Okx body dto.SubscriptionAddRequest true "Create Okx"
// @Produce application/json
// @Tags Okx
// @Success 200 {object} model.Okx
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/create/subscription [post]
func CreateSubscription(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("CreateSubscription", c.GetString(middlewares.RequestIDKey))

		var input dto.SubscriptionAddRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Msg("failed to bind request")
			return
		}

		data := model.Okx{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateSubscriptionBiz(store)

		if err := business.CreateSubscription(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Msg("failed to create Okx")
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.Okx]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Okx created successfully",
			Data:              data,
		})
		log.Info().Str("Subscription_id", data.ID.String()).Msg("Okx created successfully")
	}
}
