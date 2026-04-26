package biz

import (
	"strings"
	"testing"
)

func TestStripLLMEmphasis_boldAndItalic(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"**Lệnh:**", "Lệnh:"},
		{"a **b** c", "a b c"},
		{"*nhấn mạnh*", "nhấn mạnh"},
		{"**a** *b* __c__", "a b c"},
		{"*a* *b*", "a b"},
		{"2 * 3", "2 * 3"},
		{"**", ""},
	}
	for _, c := range cases {
		if got := StripLLMEmphasis(c.in); got != c.out {
			t.Errorf("StripLLMEmphasis(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestStripMarketDataDump_RemovesEchosNewsLine(t *testing.T) {
	in := "Chờ nhé.\nNews: USD CPI m/m in 12min (HIGH) [active]\nVì CPI sắp ra mà thôi."
	out := StripMarketDataDump(in)
	if strings.Contains(out, "News: USD CPI") {
		t.Fatalf("expected News echo stripped, got %q", out)
	}
	if !strings.Contains(out, "CPI sắp ra") {
		t.Fatalf("expected prose kept, got %q", out)
	}
}
