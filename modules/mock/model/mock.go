package model

import (
	"j-okx-ai/common"
)

const (
	EntityName = "Novel"
)

type Mock struct {
	common.BaseModel `bson:",inline"`
}

func (*Mock) CollectionName() string {
	return "Mocks"
}
