package transport

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/auth/biz"
	dto "j_ai_trade/modules/auth/model/dto"
	"j_ai_trade/utils"
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

		claims, err := utils.ParseRefreshToken(input.Token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusUnauthorized,
				Success:           false,
				Message:           "invalid or expired refresh token",
				Data:              nil,
			})
			c.Abort()
			return
		}
		userId, err := utils.GetUserIDFromClaims(claims)
		if err != nil {
			c.JSON(http.StatusUnauthorized, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusUnauthorized,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			c.Abort()
			return
		}

		business := biz.NewRefreshTokenBiz()
		accessToken, refreshToken, err := business.RefreshToken(userId)

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
