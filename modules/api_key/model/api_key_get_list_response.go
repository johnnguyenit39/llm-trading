package model

import "j-ai-trade/common"

type ApiKeyGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []ApiKey          `json:"List"`
}
