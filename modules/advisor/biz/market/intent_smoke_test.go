package market

import (
	"testing"

	"j_ai_trade/trading/models"
)

// TestDetectDefaultAndBTC: default pair is XAUUSDT; BTCUSDT only when
// the user names BTC/bitcoin/BTCUSDT. Generic "crypto" without BTC
// still falls back to XAUUSDT.
func TestDetectDefaultAndBTC(t *testing.T) {
	res := NewSymbolResolver()
	det := NewIntentDetector(res)

	cases := []struct {
		name    string
		text    string
		wantSym string
		wantTF  models.Timeframe
	}{
		{"explicit symbol", "XAUUSDT giá bao nhiêu", "XAUUSDT", models.TF_M15},
		{"vietnamese alias", "vàng đang sao rồi", "XAUUSDT", models.TF_M15},
		{"ascii-folded alias", "vang thế nào", "XAUUSDT", models.TF_M15},
		{"short XAU", "xau", "XAUUSDT", models.TF_M15},
		{"english alias", "gold price now?", "XAUUSDT", models.TF_M15},
		{"bare follow-up falls back to XAUUSDT", "bây giờ bao nhiêu", "XAUUSDT", models.TF_M15},
		{"small talk still routes to XAUUSDT", "cảm ơn bạn", "XAUUSDT", models.TF_M15},
		{"explicit TF M5 honoured", "vàng thế nào M5", "XAUUSDT", models.TF_M5},
		{"explicit TF H1 honoured", "bây giờ bao nhiêu H1", "XAUUSDT", models.TF_H1},
		{"explicit TF H4 honoured", "XAU H4", "XAUUSDT", models.TF_H4},
		{"scalp keyword maps to M15", "XAU scalp", "XAUUSDT", models.TF_M15},
		{"explicit TF M1 still resolvable", "XAU M1", "XAUUSDT", models.TF_M1},
		{"explicit BTCUSDT", "BTCUSDT giá bao nhiêu", "BTCUSDT", models.TF_M15},
		{"btc alias", "btc đang sao", "BTCUSDT", models.TF_M15},
		{"bitcoin alias", "bitcoin buy hay sell", "BTCUSDT", models.TF_M15},
		{"crypto without btc still XAU", "crypto giờ thế nào", "XAUUSDT", models.TF_M15},
		{"first token wins XAU before BTC", "xau và btc", "XAUUSDT", models.TF_M15},
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

// TestParseCommand defaults to XAUUSDT; /analyze btc honours BTCUSDT.
func TestParseCommand(t *testing.T) {
	res := NewSymbolResolver()
	det := NewIntentDetector(res)

	cases := []struct {
		name    string
		text    string
		wantSym string
		wantTF  models.Timeframe
	}{
		{"bare /analyze", "/analyze", "XAUUSDT", models.TF_M15},
		{"/analyze M5", "/analyze M5", "XAUUSDT", models.TF_M5},
		{"/analyze H4", "/analyze H4", "XAUUSDT", models.TF_H4},
		{"/analyze XAU H1", "/analyze XAU H1", "XAUUSDT", models.TF_H1},
		{"/signal alias", "/signal", "XAUUSDT", models.TF_M15},
		{"/analyze btc H1", "/analyze btc H1", "BTCUSDT", models.TF_H1},
		{"/analyze bitcoin", "/analyze bitcoin", "BTCUSDT", models.TF_M15},
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
