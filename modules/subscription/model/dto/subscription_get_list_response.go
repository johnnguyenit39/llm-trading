package model

import (
	"j-ai-trade/common"
	"j-ai-trade/modules/subscription/model"
)

type SubscriptionGetListResponse struct {
	Paging common.Pagination    `json:"Paging"`
	List   []model.Subscription `json:"List"`
}
