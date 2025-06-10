package model

import (
	"j-okx-ai/common"
	model "j-okx-ai/modules/mock/model"
)

type MockGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Mock      `json:"List"`
}
