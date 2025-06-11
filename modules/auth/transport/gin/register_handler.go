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
// @Summary Register with email and password
// @Description Register with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param auth body dto.RegisterRequest true "Register"
// @Success 201 {object} common.BaseApiResponse[bool] "User created successfully"
// @Failure 400 {object} common.BaseApiResponse[any] "Bad Request"
// @Failure 500 {object} common.BaseApiResponse[any] "Internal Server Error"
// @Router /v1/auth/register [post]
func Register(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("Register", c.GetString(middlewares.RequestIDKey))

		var input dto.RegisterRequest

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
			PhoneNumber: input.PhoneNumber,
			Password:    input.Password,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewRegisterBiz(store)

		if err := business.Register(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to create user")
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[userModel.User]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "User is created successfully",
			Data:              data,
		})
		log.Info().Str("user_id", data.ID.String()).Msg("user is created successfully")
	}
}
