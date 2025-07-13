package model

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/j_bot/model"
)

type JbotGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Jbot      `json:"List"`
}
