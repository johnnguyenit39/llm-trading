package model

import (
	"j-okx-ai/common"
)

const (
	EntityName = "Novel"
)

type Okx struct {
	common.BaseModel `bson:",inline"`
}

func (*Okx) CollectionName() string {
	return "Mocks"
}
