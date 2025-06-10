package ginUser

import (
	"j-okx-ai/common"
	"j-okx-ai/logger"
	"j-okx-ai/middlewares"
	"j-okx-ai/modules/user/biz"
	"j-okx-ai/modules/user/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// DeleteUser godoc
// @Summary Delete new User
// @Description Delete a new User
// @Param id path string true "User ID" // Updated to just type string
// @Produce application/json
// @Tags User
// @Success 200 {object} model.User
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/delete/user/{id} [delete]
func DeleteUser(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("DeleteUser", c.GetString(middlewares.RequestIDKey))

		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "User ID is required",
				Data:              nil,
			})
			log.Error().Msg("User ID is required")
			return
		}

		store := storage.NewMongoDbStore(db)
		business := biz.NewDeleteUserBiz(store)

		data, err := business.DeleteUser(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Str("user_id", id).Err(err).Msg("failed to delete user")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[any]{
			Success:           data,
			HttpRequestStatus: http.StatusOK,
			Message:           "Delete user successfully",
			Data:              nil,
		})
		log.Info().Str("user_id", id).Msg("Delete user successfully")
	}
}
