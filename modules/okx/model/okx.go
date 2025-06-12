package model

import (
	"j-ai-trade/common"
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
