package model

import (
	"j_ai_trade/common"
	model "j_ai_trade/modules/okx/model"
)

type SubscriptionGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Okx       `json:"List"`
}
