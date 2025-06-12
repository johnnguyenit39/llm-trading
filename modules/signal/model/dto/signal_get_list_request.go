package model

import (
	"j-ai-trade/common"
)

type SignalGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
