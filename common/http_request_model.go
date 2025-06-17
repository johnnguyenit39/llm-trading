package common

type HttpRequestModel struct {
	URL     string            // Request URL
	Method  string            // HTTP Method (GET, POST, PUT, DELETE)
	Body    []byte            // Request body (for POST/PUT)
	Params  map[string]string // Query parameters
	Headers map[string]string // HTTP Headers
}
