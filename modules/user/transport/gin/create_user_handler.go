package ginUser

import (
	"j_ai_trade/common"
	"j_ai_trade/logger"
	"j_ai_trade/middlewares"
	"j_ai_trade/modules/user/biz"
	model "j_ai_trade/modules/user/model"
	requestModel "j_ai_trade/modules/user/model/requests"
	"j_ai_trade/modules/user/storage"
	"j_ai_trade/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateUver godoc
// @Summary Create new User
// @Description Create a new User
// @Param User body model.UserAddRequest true "Create User"
// @Produce application/json
// @Tags User
// @Success 200 {object} model.User
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/user/create [post]
func CreateUser(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("CreateUser", c.GetString(middlewares.RequestIDKey))

		var input requestModel.UserAddRequest

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

		hashedPassword, err := utils.HashPassword(input.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to hash the password")
			return
		}

		data := model.User{
			PhoneNumber: input.PhoneNumber,
			Password:    hashedPassword,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateUserBiz(store)

		if err := business.CreateUser(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to create user")
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.User]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "User is created successfully",
			Data:              data,
		})
		log.Info().Str("user_id", data.ID.String()).Msg("User is created successfully")
	}
}
