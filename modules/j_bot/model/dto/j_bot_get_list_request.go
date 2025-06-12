package model

import (
	"j-ai-trade/common"
)

type JbotGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
