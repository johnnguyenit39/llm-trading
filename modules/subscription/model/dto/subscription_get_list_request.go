package model

import (
	"j-ai-trade/common"
)

type SubscriptionGetListRequest struct {
	Pagination common.PaginationRequest `json:"Pagination"`
}
