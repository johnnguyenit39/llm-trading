package model

import (
	"j_ai_trade/common"
)

type PermissionGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
