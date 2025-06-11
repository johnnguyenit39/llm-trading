package model

import (
	"j-ai-trade/common"
)

type PermissionGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
