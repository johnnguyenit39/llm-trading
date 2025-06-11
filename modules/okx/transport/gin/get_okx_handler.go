package ginNovel

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/okx/biz"
	dto "j-ai-trade/modules/okx/model/dto"
	"j-ai-trade/modules/okx/storage"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetSubscriptions godoc
// @Summary Get a list of Okx
// @Description Retrieve a list of Subscriptions based on provided filters and pagination
// @Param page_number query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Produce json
// @Tags Okx
// @Success 200 {array} model.Okx
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/get/mocks [get]
func GetSubscriptions(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("GetSubscriptions", c.GetString(middlewares.RequestIDKey))

		pageNumber := c.DefaultQuery("page_number", "1")
		pageSize := c.DefaultQuery("page_size", "10")

		// Convert to integers
		pageNumberInt, err := strconv.Atoi(pageNumber)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page_number"})
			log.Error().Err(err).Msg("failed to convert page_number to int")
			return
		}

		pageSizeInt, err := strconv.Atoi(pageSize)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page_size"})
			log.Error().Err(err).Msg("failed to convert page_size to int")
			return
		}

		pagination := &common.Pagination{
			Size:  pageSizeInt,
			Index: pageNumberInt,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewGetSubscriptionsBiz(store)

		list, err := business.GetSubscriptions(c.Request.Context(), pagination)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to get Subscriptions")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[dto.SubscriptionGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Okx list successfully",
			Data: dto.SubscriptionGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
		log.Info().Int("count", len(list)).Msg("Get Okx list successfully")
	}
}
