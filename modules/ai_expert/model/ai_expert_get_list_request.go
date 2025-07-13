package model

import (
	"j_ai_trade/common"
)

type AiExpertGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
