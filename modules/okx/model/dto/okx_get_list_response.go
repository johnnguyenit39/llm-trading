package model

import (
	"j-okx-ai/common"
	model "j-okx-ai/modules/okx/model"
)

type MockGetListResponse struct {
	Paging common.Pagination `json:"Paging"`
	List   []model.Okx       `json:"List"`
}
