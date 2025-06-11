package model

import "j-ai-trade/common"

type PermissionGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []Permission      `json:"List"`
}
