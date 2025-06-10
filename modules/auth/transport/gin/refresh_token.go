package transport

import (
	"j-okx-ai/common"
	"j-okx-ai/modules/auth/biz"
	dto "j-okx-ai/modules/auth/model/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AuthLogin godoc
// @Summary Get new AccessToken and RefreshToken
// @Description Get new AccessToken and RefreshToken
// @Tags Authentication
// @Accept json
// @Produce json
// @Param auth body dto.RefreshTokenRequest true "Register"
// @Success 201 {object} common.BaseApiResponse[bool] "User created successfully"
// @Failure 400 {object} common.BaseApiResponse[any] "Bad Request"
// @Failure 500 {object} common.BaseApiResponse[any] "Internal Server Error"
// @Router /v1/auth/refresh-token [post]
func RefreshToken() func(*gin.Context) {
	return func(c *gin.Context) {
		userID, _ := c.Get("userID")

		var input dto.RefreshTokenRequest

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

		business := biz.NewRefreshTokenBiz()
		accessToken, refreshToken, err := business.RefreshToken(userID.(string))

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[dto.RefreshTokenResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Refresh Token successfully",
			Data: dto.RefreshTokenResponse{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			},
		})
	}
}
