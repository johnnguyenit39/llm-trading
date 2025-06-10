package ginNovel

import (
	"j-okx-ai/common"
	"j-okx-ai/logger"
	"j-okx-ai/middlewares"
	"j-okx-ai/modules/mock/biz"
	model "j-okx-ai/modules/mock/model"
	dto "j-okx-ai/modules/mock/model/dto"
	"j-okx-ai/modules/mock/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// CreateMock godoc
// @Summary Create new Mock
// @Description Create a new Mock
// @Param Mock body model.MockAddRequest true "Create Mock"
// @Produce application/json
// @Tags Mock
// @Success 200 {object} model.Mock
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

		data := model.Mock{}

		store := storage.NewMongoDbStore(db)
		business := biz.NewCreateMockBiz(store)

		if err := business.CreateMock(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Msg("failed to create Mock")
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.Mock]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Mock created successfully",
			Data:              data,
		})
		log.Info().Str("Mock_id", data.ID.Hex()).Msg("Mock created successfully")
	}
}
