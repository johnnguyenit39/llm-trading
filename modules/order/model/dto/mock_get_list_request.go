package model

import (
	"j-ai-trade/common"
)

type OrderGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
