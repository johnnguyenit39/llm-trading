package model

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/otp/model"
)

type OtpGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Otp       `json:"List"`
}
