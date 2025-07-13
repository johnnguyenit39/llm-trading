package model

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/signal/model"
)

type SignalGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Signal    `json:"List"`
}
