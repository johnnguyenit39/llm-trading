package model

import (
	"j_ai_trade/common"
)

type OrderGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
