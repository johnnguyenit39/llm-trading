package model

type ApiKeyAddRequest struct {
	// API key for OKX
	ApiKey string `json:"api_key" binding:"required"`
	// Secret key for OKX
	ApiSecret string `json:"api_secret" binding:"required"`
	// Broker type
	Broker string `json:"broker" binding:"required"`
	// Passphrase for OKX
	PassPhrase string `json:"pass_phrase" binding:"required"`
}
