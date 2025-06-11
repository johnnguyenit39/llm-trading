package ginNovel

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/okx/biz"
	dto "j-ai-trade/modules/okx/model/dto"
	"j-ai-trade/modules/okx/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// CancelOrder godoc
// @Summary Cancel an existing order
// @Description Cancel an order by its ID
// @Accept json
// @Produce application/json
// @Tags Okx
// @Param request body dto.CancelOrderRequest true "Order cancellation parameters"
// @Success 200 {object} common.BaseApiResponse[any]
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/cancel/order [post]
func CancelOrder(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {
		var req dto.CancelOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid request format: " + err.Error(),
				Data:              nil,
			})
			return
		}

		// Validate request
		if req.OrderID == "" {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Order ID is required",
				Data:              nil,
			})
			return
		}

		store := storage.NewMongoDbStore(db)
		business := biz.NewCancelOrderBiz(store)

		response, err := business.CancelOrder(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusInternalServerError,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[string]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Order cancelled successfully",
			Data:              string(response),
		})
	}
}
