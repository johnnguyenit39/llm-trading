package ginOtp

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/otp/biz"
	"j-ai-trade/modules/otp/model"
	dto "j-ai-trade/modules/otp/model/dto"
	"j-ai-trade/modules/otp/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateOtp godoc
// @Summary Update Otp
// @Description Update Otp
// @Param id path string true "User UUID" format(uuid)
// @Param user body model.OtpUpdateRequest true "Update User"  // Correctly specify the request body
// @Produce application/json
// @Tags Otp
// @Success 200 {object} dto.OtpGetResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/otp/update [put]
func UpdateOtp(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input dto.OtpUpdateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		var data model.Otp
		store := storage.NewPostgresStore(db)
		business := biz.NewUpdateOtpBiz(store)
		business.UpdateOtp(c.Request.Context(), id, &data)

		if err := business.UpdateOtp(c.Request.Context(), id, &data); err != nil {
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
			Message:           "User updated successfully",
			Data:              response,
		})
	}
}
