package ginOtp

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/otp/biz"
	dto "j-ai-trade/modules/otp/model/dto"
	"j-ai-trade/modules/otp/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetOtp godoc
// @Summary Get Otp
// @Description Return Otp
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Otp
// @Success 200 {object} dto.OtpGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/otp/get/{id} [get]
func GetOtpById(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		store := storage.NewPostgresStore(db)
		business := biz.NewGetOtpByIdBiz(store)
		_, err := business.GetOtpById(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		response := dto.OtpGetResponse{}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.OtpGetResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Otp user successfully",
			Data:              response,
		})
	}
}
