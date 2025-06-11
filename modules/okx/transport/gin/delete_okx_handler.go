package ginNovel

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/okx/biz"
	"j-ai-trade/modules/okx/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// DeleteMock godoc
// @Summary Delete new Okx
// @Description Delete a new Okx
// @Param id path string true "Okx ID" // Updated to just type string
// @Produce application/json
// @Tags Okx
// @Success 200 {object} model.Okx
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/delete/okx/{id} [delete]
func DeleteMock(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("DeleteMock", c.GetString(middlewares.RequestIDKey))

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
		business := biz.NewDeleteMockBiz(store)

		data, err := business.DeleteMock(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Str("Mock_id", id).Msg("failed to delete Okx")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[any]{
			Success:           data,
			HttpRequestStatus: http.StatusOK,
			Message:           "Delete Okx successfully",
			Data:              nil,
		})
		log.Info().Str("Mock_id", id).Msg("Delete Okx successfully")
	}
}
