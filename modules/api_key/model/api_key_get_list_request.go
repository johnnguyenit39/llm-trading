package model

import (
	"j_ai_trade/common"
)

type ApiKeyGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
