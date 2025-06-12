package base64

import (
	"encoding/base64"
	"fmt"
	"io"
	"j-ai-trade/appi18n"
	"net/http"
	"strings"
)

func ExtractImageMetaData(base64String string) (string, string, error) {
	if strings.HasPrefix(base64String, "data:") && strings.Contains(base64String, ";base64,") {
		parts := strings.SplitN(base64String, ",", 2)
		if len(parts) == 2 {
			metaParts := strings.SplitN(parts[0], ":", 2)
			if len(metaParts) == 2 {
				return metaParts[1], parts[1], nil
			}
		}
	}
	fmt.Printf("invalid base64 data URI format\n")
	return "", "", fmt.Errorf(appi18n.InvalidBase64DataUriFormat)
}

func FetchImageFromURLToBase64(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	contentType := http.DetectContentType(imageData)
	base64String := base64.StdEncoding.EncodeToString(imageData)
	return "data:" + contentType + ";base64," + base64String
}
