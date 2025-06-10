package ginNovel

import (
	"j-okx-ai/common"
	"j-okx-ai/modules/okx/biz"
	dto "j-okx-ai/modules/okx/model/dto"
	"j-okx-ai/modules/okx/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetMock godoc
// @Summary Get Okx
// @Description Return Okx
// @Produce application/json
// @Tags Okx
// @Success 200 {object} model.Okx
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/get/okx-info [get]
func GetOkxInfo(db *mongo.Database) func(*gin.Context) {
	return func(c *gin.Context) {
		store := storage.NewMongoDbStore(db)
		business := biz.NewGetOkxInfoBiz(store)

		data, err := business.GetOkxInfo(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusOK, common.BaseApiResponse[dto.OkxInfoResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Okx info successfully",
			Data:              *data,
		})
	}
}
