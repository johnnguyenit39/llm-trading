package ginUser

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/user/biz"
	model "j-ai-trade/modules/user/model"
	requestModel "j-ai-trade/modules/user/model/requests"
	"j-ai-trade/modules/user/storage"
	"j-ai-trade/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
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
// @Router /v1/create/user [post]
func CreateUser(db *mongo.Database) func(*gin.Context) {
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

		store := storage.NewMongoDbStore(db)
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
		log.Info().Str("user_id", data.ID.Hex()).Msg("User is created successfully")
	}
}
