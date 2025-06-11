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
	"go.mongodb.org/mongo-driver/mongo"
)

// CreateMock godoc
// @Summary Create new Okx
// @Description Create a new Okx
// @Param Okx body model.MockAddRequest true "Create Okx"
// @Produce application/json
// @Tags Okx
// @Success 200 {object} model.Okx
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/create/mock [post]
func CreateMock(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("CreateMock", c.GetString(middlewares.RequestIDKey))

		var input dto.MockAddRequest
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

		store := storage.NewMongoDbStore(db)
		business := biz.NewCreateMockBiz(store)

		if err := business.CreateMock(c.Request.Context(), &data); err != nil {
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
		log.Info().Str("Mock_id", data.ID.Hex()).Msg("Okx created successfully")
	}
}
