package ginSignal

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/signal/biz"
	dto "j-ai-trade/modules/signal/model/dto"
	"j-ai-trade/modules/signal/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetSignal godoc
// @Summary Get Signal
// @Description Return Signal
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Signal
// @Success 200 {object} dto.SignalGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/signal/get [get]
func GetSignalById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetSignalByIdBiz(store)
		_, err := business.GetSignalById(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := dto.SignalGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.SignalGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Signal user successfully",
			Data:              response,
		})
	}
}
