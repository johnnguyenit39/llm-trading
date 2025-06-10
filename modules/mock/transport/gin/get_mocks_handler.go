package ginNovel

import (
	"j-okx-ai/common"
	"j-okx-ai/logger"
	"j-okx-ai/middlewares"
	"j-okx-ai/modules/mock/biz"
	dto "j-okx-ai/modules/mock/model/dto"
	"j-okx-ai/modules/mock/storage"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetMocks godoc
// @Summary Get a list of Mock
// @Description Retrieve a list of Mocks based on provided filters and pagination
// @Param page_number query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Produce json
// @Tags Mock
// @Success 200 {array} model.Mock
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/Get/Mocks [get]
func GetMocks(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("GetMocks", c.GetString(middlewares.RequestIDKey))

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
			PageSize:   pageSizeInt,
			PageNumber: pageNumberInt,
		}

		store := storage.NewMongoDbStore(db)
		business := biz.NewGetMocksBiz(store)

		list, err := business.GetMocks(c.Request.Context(), pagination)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to get Mocks")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[dto.MockGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Mock list successfully",
			Data: dto.MockGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
		log.Info().Int("count", len(list)).Msg("Get Mock list successfully")
	}
}
