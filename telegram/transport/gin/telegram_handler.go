package ginTelegram

import (
	"j_ai_trade/common"
	"j_ai_trade/telegram"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SendTelegramMessage godoc
// @Summary Send a message to Telegram channel
// @Description Send a message to the configured Telegram channel
// @Tags Telegram
// @Accept json
// @Produce json
// @Param request body telegram.SendMessageRequest true "Message details"
// @Success 200 {object} common.BaseApiResponse[bool]
// @Failure 400 {object} common.BaseApiResponse[any]
// @Router /v1/telegram/send [post]
func SendTelegramMessage(telegramService *telegram.TelegramService) func(*gin.Context) {
	return func(c *gin.Context) {
		var input telegram.SendMessageRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		err := telegramService.SendMessage(input.Text)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[bool]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Message sent successfully",
			Data:              true,
		})
	}
}
