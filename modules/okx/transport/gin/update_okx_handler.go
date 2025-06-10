package ginNovel

import (
	"j-okx-ai/common"
	"j-okx-ai/logger"
	"j-okx-ai/middlewares"
	"j-okx-ai/modules/okx/biz"
	model "j-okx-ai/modules/okx/model"
	dto "j-okx-ai/modules/okx/model/dto"
	"j-okx-ai/modules/okx/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// UpdateMock godoc
// @Summary Update Okx
// @Description Update Okx
// @Param id path string true "Okx ID"
// @Param Okx body model.MockUpdateRequest true "Update Okx"
// @Produce application/json
// @Tags Okx
// @Success 200 {object} model.Okx
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/update/okx/{id} [put]
func UpdateMock(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("UpdateMock", c.GetString(middlewares.RequestIDKey))
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

		var input dto.MockUpdateRequest
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
		business := biz.NewUpdateMockBiz(store)

		if err := business.UpdateMock(c.Request.Context(), id, &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Str("request_id", c.GetString(middlewares.RequestIDKey)).Msg("failed to update Okx")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[model.Okx]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Okx is updated successfully",
			Data:              data,
		})
		log.Info().Str("Mock_id", id).Msg("Okx is updated successfully")
	}
}
