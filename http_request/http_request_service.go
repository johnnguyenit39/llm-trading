package httprequest

import (
	httpclient "j_ai_trade/common"
	"net/http"
)

type HttpService struct {
	repo HttpRequestRepository
}

func NewHttpService(repo HttpRequestRepository) *HttpService {
	return &HttpService{repo: repo}
}

func (s *HttpService) Get(url string, params map[string]string, headers map[string]string) (*http.Response, error) {
	request := httpclient.HttpRequestModel{
		URL:     url,
		Method:  http.MethodGet,
		Params:  params,
		Headers: headers,
	}
	return s.repo.DoRequest(request)
}

func (s *HttpService) Post(url string, body []byte, headers map[string]string) (*http.Response, error) {
	request := httpclient.HttpRequestModel{
		URL:     url,
		Method:  http.MethodPost,
		Body:    body,
		Headers: headers,
	}
	return s.repo.DoRequest(request)
}

func (s *HttpService) Put(url string, body []byte, headers map[string]string) (*http.Response, error) {
	request := httpclient.HttpRequestModel{
		URL:     url,
		Method:  http.MethodPut,
		Body:    body,
		Headers: headers,
	}
	return s.repo.DoRequest(request)
}

func (s *HttpService) Delete(url string, body []byte, headers map[string]string) (*http.Response, error) {
	request := httpclient.HttpRequestModel{
		URL:     url,
		Method:  http.MethodDelete,
		Body:    body,
		Headers: headers,
	}
	return s.repo.DoRequest(request)
}
