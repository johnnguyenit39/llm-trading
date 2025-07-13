package model

import (
	"j_ai_trade/common"
)

type OtpGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
