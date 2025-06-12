package ginSignal

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/signal/biz"
	"j-ai-trade/modules/signal/model"
	dto "j-ai-trade/modules/signal/model/dto"
	"j-ai-trade/modules/signal/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateSignal godoc
// @Summary Create new Signal
// @Description Create a new Signal
// @Param Signal body dto.SignalAddRequest true "Create Signal"
// @Produce application/json
// @Tags Signal
// @Success 200 {object} dto.SignalGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/signal/create [post]
func CreateSignal(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.SignalAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.Signal{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateSignalBiz(store)

		if err := business.CreateSignal(c.Request.Context(), &data); err != nil {
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
			Message:           "Signal created successfully",
			Data:              response,
		})
	}
}
