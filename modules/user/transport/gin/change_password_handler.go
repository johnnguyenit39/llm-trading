package ginUser

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/user/biz"
	model "j-ai-trade/modules/user/model"
	requestModel "j-ai-trade/modules/user/model/requests"
	"j-ai-trade/modules/user/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateUserPassword godoc
// @Summary Update User Password
// @Description Update User Password
// @Param id path string true "User ID"
// @Param User body model.UserUpdatePasswordRequest true "Update User"
// @Produce application/json
// @Tags User
// @Success 200 {object} model.User
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/update/password/user/{id} [put]
func UpdateUserPassword(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logger.GetLogger("UpdateUserPassword", c.GetString(middlewares.RequestIDKey))

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

		var input requestModel.UserUpdatePasswordRequest
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

		if input.NewPassword != input.NewPasswordConfirmation {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Your new password does not match.",
				Data:              nil,
			})
			return
		}

		store := storage.NewPostgresStore(db)

		business := biz.NewUpdateUserPasswordBiz(store)
		data, err := business.UpdateUserPassword(c.Request.Context(), id, &input)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Str("user_id", id).Msg("failed to update user")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[model.User]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "User's password is updated successfully",
			Data:              *data,
		})
		log.Info().Str("user_id", id).Msg("User's password is updated successfully")
	}
}
