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

// GetSignals godoc
// @Summary Get a list of Signal
// @Description Retrieve a list of Signals based on provided filters and pagination
// @Produce json
// @Tags Signal
// @Param Signal body dto.SignalGetListRequest true "Get Signals"  // Correctly specify the request body
// @Success 200 {object} dto.SignalGetListResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/signal/list [post]
func GetSignals(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.SignalGetListRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		pagination := &common.Pagination{
			Size:  input.Pagination.Size,
			Index: input.Pagination.Index,
		}
		store := storage.NewPostgresStore(db)
		business := biz.NewGetSignalsBiz(store)
		list, err := business.GetSignals(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.SignalGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Signal list successfully",
			Data: dto.SignalGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
