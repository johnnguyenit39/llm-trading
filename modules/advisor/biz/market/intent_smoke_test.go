package market

import (
	"testing"

	"j_ai_trade/trading/models"
)

// TestDetectGoldOnly pins down the gold-only behaviour: every
// non-empty message resolves to XAUUSDT — either via an explicit
// alias ("vàng", "XAU", "gold", "XAUUSDT") or via DefaultSymbol when
// the user didn't name anything. The optional timeframe is honoured
// when present; otherwise we default to M1 (scalping entry TF).
func TestDetectGoldOnly(t *testing.T) {
	res := NewSymbolResolver()
	det := NewIntentDetector(res)

	cases := []struct {
		name    string
		text    string
		wantSym string
		wantTF  models.Timeframe
	}{
		{"explicit symbol", "XAUUSDT giá bao nhiêu", "XAUUSDT", models.TF_M1},
		{"vietnamese alias", "vàng đang sao rồi", "XAUUSDT", models.TF_M1},
		{"ascii-folded alias", "vang thế nào", "XAUUSDT", models.TF_M1},
		{"short XAU", "xau", "XAUUSDT", models.TF_M1},
		{"english alias", "gold price now?", "XAUUSDT", models.TF_M1},
		{"bare follow-up falls back to XAUUSDT", "bây giờ bao nhiêu", "XAUUSDT", models.TF_M1},
		{"small talk still routes to XAUUSDT", "cảm ơn bạn", "XAUUSDT", models.TF_M1},
		{"explicit TF M5 honoured", "vàng thế nào M5", "XAUUSDT", models.TF_M5},
		{"explicit TF H1 honoured", "bây giờ bao nhiêu H1", "XAUUSDT", models.TF_H1},
		{"explicit TF H4 honoured", "XAU H4", "XAUUSDT", models.TF_H4},
		{"scalp keyword maps to M1", "XAU scalp", "XAUUSDT", models.TF_M1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := det.DetectWithFallback(tc.text, "")
			if got.Symbol != tc.wantSym {
				t.Fatalf("symbol: want %q, got %q", tc.wantSym, got.Symbol)
			}
			if got.Timeframe != tc.wantTF {
				t.Fatalf("tf: want %q, got %q", tc.wantTF, got.Timeframe)
			}
		})
	}
}

// TestParseCommand_GoldOnly confirms /analyze [TF] defaults to
// XAUUSDT on M1, and that an explicit TF is preserved.
func TestParseCommand_GoldOnly(t *testing.T) {
	res := NewSymbolResolver()
	det := NewIntentDetector(res)

	cases := []struct {
		name    string
		text    string
		wantSym string
		wantTF  models.Timeframe
	}{
		{"bare /analyze", "/analyze", "XAUUSDT", models.TF_M1},
		{"/analyze M5", "/analyze M5", "XAUUSDT", models.TF_M5},
		{"/analyze H4", "/analyze H4", "XAUUSDT", models.TF_H4},
		{"/analyze XAU H1", "/analyze XAU H1", "XAUUSDT", models.TF_H1},
		{"/signal alias", "/signal", "XAUUSDT", models.TF_M1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := det.ParseCommand(tc.text)
			if !got.Explicit {
				t.Fatalf("expected Explicit=true for command input")
			}
			if got.Symbol != tc.wantSym {
				t.Fatalf("symbol: want %q, got %q", tc.wantSym, got.Symbol)
			}
			if got.Timeframe != tc.wantTF {
				t.Fatalf("tf: want %q, got %q", tc.wantTF, got.Timeframe)
			}
		})
	}
}
