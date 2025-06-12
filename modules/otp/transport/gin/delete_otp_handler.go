package ginOtp

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/otp/biz"
	"j-ai-trade/modules/otp/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeleteOtp godoc
// @Summary Delete new Otp
// @Description Delete a new Otp
// @Param id path string true "User UUID" format(uuid)
// @Produce application/json
// @Tags Otp
// @Success 200 {object} model.Otp
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/otp/delete [delete]
func DeleteOtp(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := storage.NewPostgresStore(db)
		business := biz.NewDeleteOtpBiz(store)
		data, err := business.DeleteOtp(c.Request.Context(), id)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[any]{
			Success:           data,
			HttpRequestStatus: http.StatusOK,
			Message:           "Delete user successfully",
			Data:              nil,
		})
	}
}
