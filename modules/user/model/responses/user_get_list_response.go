package model

import (
	"j-ai-trade/common"
	model "j-ai-trade/modules/user/model"
)

type UserGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.User      `json:"List"`
}
