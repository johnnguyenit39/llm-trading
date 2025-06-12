package model

import "j-ai-trade/common"

type AiExpertGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []AiExpert           `json:"List"`
}
