package model

import (
	"j-ai-trade/common"
)

type ApiKeyGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
