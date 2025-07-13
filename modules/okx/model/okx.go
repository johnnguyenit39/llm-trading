package model

import (
	"j_ai_trade/common"
)

const (
	EntityName = "Novel"
)

type Okx struct {
	common.BaseModel `bson:",inline"`
}

func (*Okx) CollectionName() string {
	return "subscriptions"
}
