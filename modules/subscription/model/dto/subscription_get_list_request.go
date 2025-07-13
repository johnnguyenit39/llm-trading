package model

import (
	"j_ai_trade/common"
)

type SubscriptionGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
