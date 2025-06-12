package model

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/order/model"
)

type OrderGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Order     `json:"List"`
}
