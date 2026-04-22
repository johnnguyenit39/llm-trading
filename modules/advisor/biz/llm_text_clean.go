package biz

import (
	"regexp"
	"strings"
)

var singleEmphasis = regexp.MustCompile(`\*([^*\n]+)\*`)

// StripLLMEmphasis removes Markdown-style emphasis markers models often
// emit (**bold**, *italic*, __underline__) so Telegram shows plain
// Vietnamese text instead of asterisk noise. Safe to run on the full
// reply; fenced ``` code blocks are rare in the same string as labels
// like **Lệnh:** in practice. If a fence contains literal **, this may
// alter it — the advisor path only fences strict JSON, which we parse
// before stripping for history.
func StripLLMEmphasis(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	prev := ""
	for prev != s {
		prev = s
		s = singleEmphasis.ReplaceAllString(s, "$1")
	}
	return s
}
