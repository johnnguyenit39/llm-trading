package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPRequest struct {
	URL     string
	Method  string
	Body    any
	Headers map[string]string
}

type HTTPClient struct {
	client  *http.Client
	baseURL string
}

func NewHTTPClient(baseURL string) *HTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip TLS verification
			MinVersion:         tls.VersionTLS12,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Minute,
	}

	return &HTTPClient{
		client:  client,
		baseURL: baseURL,
	}
}

func (c *HTTPClient) Send(ctx context.Context, req HTTPRequest, result interface{}) error {
	url := c.baseURL + req.URL
	var bodyReader io.Reader

	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return ErrInternal(fmt.Errorf("failed to marshal request body: %w", err))
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return ErrInternal(fmt.Errorf("failed to create request: %w", err))
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return ErrInternal(fmt.Errorf("failed to send request: %w", err))
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrInternal(fmt.Errorf("failed to read response body: %w", err))
	}

	// Try to unmarshal the response body into the result type
	if err := json.Unmarshal(body, result); err != nil {
		return ErrInternal(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return nil
}
