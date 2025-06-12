package appi18n

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const (
	UserRegisteredSuccessfully = "user_registered_successfully"
)

//go:embed active.*.json
var localeFS embed.FS

var Bundle *i18n.Bundle

func Init() {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load all translation files
	files := []string{"active.en.json", "active.vi.json"}

	for _, file := range files {
		_, err := Bundle.LoadMessageFileFS(localeFS, file)
		if err != nil {
			panic(fmt.Sprintf("load i18n file failed: %v", err))
		}
	}
}

func Translate(languageID string, messageID string, data map[string]interface{}) string {
	localizer := i18n.NewLocalizer(Bundle, languageID)

	message, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})

	if err != nil {
		return messageID
	}
	return message
}
