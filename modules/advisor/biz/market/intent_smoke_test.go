package market

import (
	"testing"

	"j_ai_trade/trading/models"
)

// TestDetectWithFallback pins down the "follow-up carries the last
// symbol" behaviour that prevents the bot from echoing stale prices
// when the user asks "bây giờ bao nhiêu" without re-mentioning BTC.
func TestDetectWithFallback(t *testing.T) {
	res := NewSymbolResolver()
	det := NewIntentDetector(res)

	cases := []struct {
		name       string
		text       string
		lastSymbol string
		wantSym    string
		wantTF     models.Timeframe
	}{
		{"explicit symbol wins over pin", "BTC giá bao nhiêu", "ETHUSDT", "BTCUSDT", models.TF_M15},
		{"bare follow-up uses pin", "bây giờ bao nhiêu", "BTCUSDT", "BTCUSDT", models.TF_M15},
		{"bare follow-up short form uses pin", "giờ thì sao", "XAUUSDT", "XAUUSDT", models.TF_M15},
		{"english follow-up uses pin", "price now?", "ETHUSDT", "ETHUSDT", models.TF_M15},
		{"follow-up without pin misses", "bây giờ bao nhiêu", "", "", ""},
		{"noise without keyword misses even with pin", "cảm ơn bạn", "BTCUSDT", "", ""},
		{"switch to ETH when user names it explicitly", "ETH thế nào", "BTCUSDT", "ETHUSDT", models.TF_M15},
		{"follow-up + explicit TF keeps the TF", "bây giờ bao nhiêu H4", "BTCUSDT", "BTCUSDT", models.TF_H4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := det.DetectWithFallback(tc.text, tc.lastSymbol)
			if got.Symbol != tc.wantSym {
				t.Fatalf("symbol: want %q, got %q", tc.wantSym, got.Symbol)
			}
			if tc.wantSym != "" && got.Timeframe != tc.wantTF {
				t.Fatalf("tf: want %q, got %q", tc.wantTF, got.Timeframe)
			}
		})
	}
}
