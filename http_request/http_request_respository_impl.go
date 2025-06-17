package httprequest

import (
	"bytes"
	"j-ai-trade/common"
	"net/http"
	"net/url"
)

type HttpRepositoryImpl struct {
	client *http.Client
}

func NewHttpRepository() HttpRequestRepository {
	return &HttpRepositoryImpl{
		client: &http.Client{},
	}
}

func (r *HttpRepositoryImpl) DoRequest(request common.HttpRequestModel) (*http.Response, error) {
	// Build the URL with query parameters
	reqURL, err := url.Parse(request.URL)
	if err != nil {
		return nil, err
	}

	query := reqURL.Query()
	for key, value := range request.Params {
		query.Set(key, value)
	}
	reqURL.RawQuery = query.Encode()

	// Create the HTTP request
	req, err := http.NewRequest(request.Method, reqURL.String(), bytes.NewBuffer(request.Body))
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	// Perform the request
	return r.client.Do(req)
}
