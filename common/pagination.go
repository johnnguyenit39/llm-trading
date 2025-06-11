package common

type Pagination struct {
	Count int64 `json:"count"`
	Index int   `json:"index"`
	Size  int   `json:"size"`
}
