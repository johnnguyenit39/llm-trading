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

// CreateOrder godoc
// @Summary Create a new order
// @Description Create a new order with specified currency and USDT pair
// @Accept json
// @Produce application/json
// @Tags Okx
// @Param request body dto.CreateOrderRequest true "Order creation parameters"
// @Success 200 {object} common.BaseApiResponse[any]
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/create/order [post]
func CreateOrder(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {
		var req dto.CreateOrderRequest
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
		if req.Currency == "" {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Currency is required",
				Data:              nil,
			})
			return
		}

		if req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Amount must be greater than 0",
				Data:              nil,
			})
			return
		}

		if req.Type == "limit" && req.Price <= 0 {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Price must be greater than 0 for limit orders",
				Data:              nil,
			})
			return
		}

		store := storage.NewMongoDbStore(db)
		business := biz.NewCreateOrderBiz(store)

		response, err := business.CreateOrder(c.Request.Context(), &req)
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
			Message:           "Order created successfully",
			Data:              string(response),
		})
	}
}
