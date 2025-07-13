package model

import "j_ai_trade/common"

type ApiKeyUpdateRequest struct {
	Broker common.Broker `json:"broker" example:"okx" binding:"required" swaggertype:"string" enums:"okx"`
	// API key for OKX
	ApiKey string `json:"api_key"`
	// Secret key for OKX
	ApiSecret string `json:"api_secret"`
	// Passphrase for OKX
	PassPhrase string `json:"pass_phrase"`
}
