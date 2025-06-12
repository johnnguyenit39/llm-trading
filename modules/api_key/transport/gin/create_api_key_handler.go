package ginApiKey

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/api_key/biz"
	"j-ai-trade/modules/api_key/model"
	"j-ai-trade/modules/api_key/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateApiKey godoc
// @Summary Create new ApiKey
// @Description Create a new ApiKey
// @Param request body model.ApiKeyAddRequest true "Create ApiKey"
// @Produce application/json
// @Tags ApiKey
// @Success 200 {boolean} true
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/create/api-key [post]
func CreateApiKey(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input model.ApiKeyAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "User ID not found",
				Data:              nil,
			})
			return
		}

		parsedUUID, err := uuid.Parse(userID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.ApiKey{
			UserID:     parsedUUID,
			ApiKey:     input.ApiKey,
			ApiSecret:  input.ApiSecret,
			PassPhrase: input.PassPhrase,
			Broker:     input.Broker,
		}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateApiKeyBiz(store)

		if err := business.CreateApiKey(c.Request.Context(), &data); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := model.ApiKeyGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[model.ApiKeyGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "ApiKey created successfully",
			Data:              response,
		})
	}
}
