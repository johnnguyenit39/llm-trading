package utils

import "strings"

func ConvertPair(pair string) string {
	return strings.ReplaceAll(pair, "/", "")
}
