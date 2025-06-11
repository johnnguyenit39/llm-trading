package model

import (
	"j-ai-trade/common"
	model "j-ai-trade/modules/okx/model"
)

type SubscriptionGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Okx       `json:"List"`
}
