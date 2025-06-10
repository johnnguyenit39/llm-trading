package ginNovel

import (
	"j-okx-ai/common"
	"j-okx-ai/logger"
	"j-okx-ai/middlewares"
	"j-okx-ai/modules/mock/biz"
	model "j-okx-ai/modules/mock/model"
	"j-okx-ai/modules/mock/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetMock godoc
// @Summary Get Mock
// @Description Return Mock
// @Param id path string true "Mock ID" // Updated to just type string
// @Produce application/json
// @Tags Mock
// @Success 200 {object} model.Mock
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/Get/Mock/{id} [get]
func GetMockById(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("GetMockById", c.GetString(middlewares.RequestIDKey))

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

		store := storage.NewMongoDbStore(db)
		business := biz.NewGetMockByIdBiz(store)

		data, err := business.GetMockById(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Str("Mock_id", id).Msg("failed to get Mock by id")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[model.Mock]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Mock user successfully",
			Data:              *data,
		})
		log.Info().Str("Mock_id", id).Msg("get Mock by id successfully")
	}
}
