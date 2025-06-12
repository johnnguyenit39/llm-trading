package model

import (
	"j-ai-trade/common"
)

type AiExpertGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
