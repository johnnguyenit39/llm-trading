package strings

import (
	"fmt"
	"j-ai-trade/helpers/email/model"
	"path"
	"strings"
)

func TruncateNotificationContent(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

func ReplaceSubject(template string, contentMappers []model.DataMapper) string {
	for _, mapper := range contentMappers {
		if mapper.Type == model.DataMapperTypeText {
			placeholder := fmt.Sprintf("{{%s}}", mapper.Field)
			template = strings.Replace(template, placeholder, mapper.Value, -1)
		}
	}
	return template
}

func RemoveExtension(filename string) string {
	ext := path.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}

func ReplaceContent(template string, contentMappers []model.DataMapper) string {
	for _, mapper := range contentMappers {
		placeholder := fmt.Sprintf("{{%s}}", mapper.Field)
		if mapper.Type == model.DataMapperTypeImageUrl {
			template = strings.Replace(template, placeholder, fmt.Sprintf(`<img src="cid:%s"/>`, mapper.Field), -1)
		} else {
			template = strings.Replace(template, placeholder, mapper.Value, -1)
		}
	}
	return template
}
