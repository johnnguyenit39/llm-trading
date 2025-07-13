package model

import (
	"j_ai_trade/common"
	model "j_ai_trade/modules/user/model"
)

type UserGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.User      `json:"List"`
}
