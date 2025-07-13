package ginJbot

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/j_bot/biz"
	"j_ai_trade/modules/j_bot/model"
	dto "j_ai_trade/modules/j_bot/model/dto"
	"j_ai_trade/modules/j_bot/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateJbot godoc
// @Summary Create new Jbot
// @Description Create a new Jbot
// @Param Jbot body dto.JbotAddRequest true "Create Jbot"
// @Produce application/json
// @Tags Jbot
// @Success 200 {object} dto.JbotGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/jbot/create [post]
func CreateJbot(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.JbotAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.Jbot{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateJbotBiz(store)

		if err := business.CreateJbot(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := dto.JbotGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.JbotGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Jbot created successfully",
			Data:              response,
		})
	}
}
