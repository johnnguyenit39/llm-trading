package httprequest

import (
	"j-ai-trade/common"
	"net/http"
)

type HttpRequestRepository interface {
	DoRequest(request common.HttpRequestModel) (*http.Response, error)
}
