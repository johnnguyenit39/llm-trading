package model

import (
	"j-ai-trade/common"
)

const (
	EntityName = "Author"
)

type User struct {
	common.BaseModel `bson:",inline"`
	PhoneNumber      string `bson:"phone_number" json:"phone_number"`
	Password         string `bson:"password" json:"-"`
}

func (*User) CollectionName() string {
	return "Users"
}
