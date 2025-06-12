package transport

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/auth/biz"
	dto "j-ai-trade/modules/auth/model/dto"
	"j-ai-trade/modules/auth/storage"
	userModel "j-ai-trade/modules/user/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthLogin godoc
// @Summary Verify Login with email and password
// @Description Verify Login with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param auth body dto.LoginRequest true "Verify Login"
// @Success 201 {object} common.BaseApiResponse[bool] "Logged in successfully"
// @Failure 400 {object} common.BaseApiResponse[any] "Bad Request"
// @Failure 500 {object} common.BaseApiResponse[any] "Internal Server Error"
// @Router /v1/auth/login [post]
func Login(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("Login", c.GetString(middlewares.RequestIDKey))

		var input dto.LoginRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to bind the request")
			return
		}

		data := userModel.User{
			Email:    input.Email,
			Password: input.Password,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewLoginBiz(store)

		userData, err := business.Login(c.Request.Context(), &data)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to login")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[dto.LoginResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Logged in successfully",
			Data:              *userData,
		})
		log.Info().Str("user_id", userData.User.ID.String()).Msg("logged in successfully")

	}
}
