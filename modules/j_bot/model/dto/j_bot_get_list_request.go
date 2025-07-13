package model

import (
	"j_ai_trade/common"
)

type JbotGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
