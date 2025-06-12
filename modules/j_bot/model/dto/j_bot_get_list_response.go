package model

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/j_bot/model"
)

type JbotGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Jbot      `json:"List"`
}
