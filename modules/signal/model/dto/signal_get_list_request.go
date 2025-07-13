package model

import (
	"j_ai_trade/common"
)

type SignalGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
