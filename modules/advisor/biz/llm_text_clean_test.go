package biz

import "testing"

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
