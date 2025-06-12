package model

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/signal/model"
)

type SignalGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Signal    `json:"List"`
}
