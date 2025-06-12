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

// UpdateSignal godoc
// @Summary Update Signal
// @Description Update Signal
// @Param id path string true "User UUID" format(uuid)
// @Param user body dto.SignalUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags Signal
// @Success 200 {object} dto.SignalGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/signal/update [put]
func UpdateSignal(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input dto.SignalUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.Signal
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdateSignalBiz(store)
		business.UpdateSignal(c.Request.Context(), id, &data)

		if err := business.UpdateSignal(c.Request.Context(), id, &data); err != nil {
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
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
