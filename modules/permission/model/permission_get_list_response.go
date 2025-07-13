package model

import "j_ai_trade/common"

type PermissionGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []Permission      `json:"List"`
}
