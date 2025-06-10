package model

type UserAddRequest struct {
	PhoneNumber string `bson:"phone_number" json:"phone_number" validate:"required"`
	Password    string `bson:"password" json:"password" validate:"required"`
}
