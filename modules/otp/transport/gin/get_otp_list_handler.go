package ginOtp

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/otp/biz"
	dto "j_ai_trade/modules/otp/model/dto"
	"j_ai_trade/modules/otp/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetOtps godoc
// @Summary Get a list of Otp
// @Description Retrieve a list of Otps based on provided filters and pagination
// @Produce json
// @Tags Otp
// @Param Otp body model.OtpGetListRequest true "Get Otps"  // Correctly specify the request body
// @Success 200 {object} dto.OtpGetListResponse
// @securityDefinitions.apiKey token
// @in header
// @name Authorization
// @Security Bearer
// @Router /v1/otp/list [post]
func GetOtps(db *gorm.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var input dto.OtpGetListRequest

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		pagination := &common.Pagination{
			Size:  input.Pagination.Size,
			Index: input.Pagination.Index,
		}
		store := storage.NewPostgresStore(db)
		business := biz.NewGetOtpsBiz(store)
		list, err := business.GetOtps(c.Request.Context(), pagination)

		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			return
		}

		c.JSON(http.StatusCreated, common.BaseApiResponse[dto.OtpGetListResponse]{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Get Otp list successfully",
			Data: dto.OtpGetListResponse{
				List:   list,
				Paging: *pagination,
			},
		})
	}
}
