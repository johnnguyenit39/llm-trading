package common

type Pagination struct {
	Count      int64 `json:"count"`
	PageNumber int   `json:"page_number"`
	PageSize   int   `json:"page_size"`
}
