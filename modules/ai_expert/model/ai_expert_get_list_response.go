package model

import "j_ai_trade/common"

type AiExpertGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []AiExpert        `json:"List"`
}
