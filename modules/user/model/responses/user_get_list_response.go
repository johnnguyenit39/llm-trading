package model

import (
	"j-okx-ai/common"
	model "j-okx-ai/modules/user/model"
)

type UserGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.User      `json:"List"`
}
