package model

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/order/model"
)

type OrderGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Order     `json:"List"`
}
