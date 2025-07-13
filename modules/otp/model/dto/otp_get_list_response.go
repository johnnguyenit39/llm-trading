package model

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/otp/model"
)

type OtpGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Otp       `json:"List"`
}
