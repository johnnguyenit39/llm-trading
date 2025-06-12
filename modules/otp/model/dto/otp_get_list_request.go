package model

import (
	"j-ai-trade/common"
)

type OtpGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
