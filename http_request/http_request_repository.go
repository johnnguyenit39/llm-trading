package httprequest

import (
	"j_ai_trade/common"
	"net/http"
)

type HttpRequestRepository interface {
	DoRequest(request common.HttpRequestModel) (*http.Response, error)
}
