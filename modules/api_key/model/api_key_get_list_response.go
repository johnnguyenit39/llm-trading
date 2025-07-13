package model

import "j_ai_trade/common"

type ApiKeyGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []ApiKey          `json:"List"`
}
