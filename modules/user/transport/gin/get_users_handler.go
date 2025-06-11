package ginUser

import (
	"j-ai-trade/common"
	"j-ai-trade/logger"
	"j-ai-trade/middlewares"
	"j-ai-trade/modules/user/biz"
	responseModel "j-ai-trade/modules/user/model/responses"
	"j-ai-trade/modules/user/storage"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUser godoc
// @Summary Get a list of User
// @Description Retrieve a list of Users based on provided filters and pagination
// @Param page_number query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Produce json
// @Tags User
// @Success 200 {array} model.User
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/get/users [get]
func GetUsers(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {

		log := logger.GetLogger("GetUsers", c.GetString(middlewares.RequestIDKey))

		pageNumber := c.DefaultQuery("page_number", "1")
		pageSize := c.DefaultQuery("page_size", "10")

		// Convert to integers
		pageNumberInt, err := strconv.Atoi(pageNumber)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid page_number",
				Data:              nil,
			})
			log.Error().Err(err).Msg("Invalid page_number")
			return
		}

		pageSizeInt, err := strconv.Atoi(pageSize)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid page_size",
				Data:              nil,
			})
			log.Error().Err(err).Msg("Invalid page_size")
			return
		}

		pagination := &common.Pagination{
			Size:  pageSizeInt,
			Index: pageNumberInt,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewGetUsersBiz(store)

		list, err := business.GetUsers(c.Request.Context(), pagination)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			log.Error().Err(err).Msg("failed to get users")
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[responseModel.UserGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get User list successfully",
			Data: responseModel.UserGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
		log.Info().Int("count", len(list)).Msg("Get User list successfully")
	}
}
