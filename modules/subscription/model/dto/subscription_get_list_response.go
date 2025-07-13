package model

import (
	"j_ai_trade/common"
	"j_ai_trade/modules/subscription/model"
)

type SubscriptionGetListResponse struct {
	Paging common.Pagination    `json:"Paging"`
	List   []model.Subscription `json:"List"`
}
