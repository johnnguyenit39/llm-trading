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

// CreateOtp godoc
// @Summary Create new Otp
// @Description Create a new Otp
// @Param Otp body model.Otp true "Create Otp"
// @Produce application/json
// @Tags Otp
// @Success 200 {object} model.Otp
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v2/otp/create [post]
func CreateOtp(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.OtpAddRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		data := model.Otp{}

		store := storage.NewPostgresStore(db)
		business := biz.NewCreateOtpBiz(store)

		if err := business.CreateOtp(c.Request.Context(), &data); err != nil {
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
			Message:           "Otp created successfully",
			Data:              response,
		})
	}
}
