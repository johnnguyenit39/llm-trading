package common

type PaginationRequest struct {
	Index int `json:"index"`
	Size  int `json:"size"`
}
