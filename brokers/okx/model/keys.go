package model

type OkxApiKeysModel struct {
	ApiKey     string `json:"api_key,omitempty"`
	ApiSecret  string `json:"api_secret,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}
