package ginUser

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/user/biz"
	model "j-ai-trade/modules/user/model"
	"j-ai-trade/modules/user/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUser godoc
// @Summary Get User
// @Description Return User
// @Param id path string true "User ID" // Updated to just type string
// @Produce application/json
// @Tags User
// @Success 200 {object} model.User
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/user/get/{id} [get]
func GetUserById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("GetUserById", c.GetString(middlewares.RequestIDKey))

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

		store := storage.NewPostgresStore(db)
		business := biz.NewGetUserByIdBiz(store)

		data, err := business.GetUserById(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Str("user_id", id).Msg("failed to get user")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[model.User]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "User user successfully",
			Data:              *data,
		})
		log.Info().Str("user_id", id).Msg("get user successfully")
	}
}
