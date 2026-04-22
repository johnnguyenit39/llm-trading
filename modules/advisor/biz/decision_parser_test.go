package biz

import (
	"strings"
	"testing"
)

func TestExtractDecision_Valid(t *testing.T) {
	reply := "Ok setup BTC M15 ổn: EMA cross lên, RSI 55, H1 cũng TREND_UP. Mình vào BUY.\n\n" +
		"```json\n" +
		`{"action":"BUY","symbol":"BTCUSDT","entry":75820.5,"stop_loss":75400,"take_profit":76800,"lot":0.01}` + "\n" +
		"```"
	got := ExtractDecision(reply)
	if got == nil {
		t.Fatalf("expected payload, got nil")
	}
	if got.Action != "BUY" || got.Symbol != "BTCUSDT" {
		t.Fatalf("bad normalisation: %+v", got)
	}
	if got.Entry != 75820.5 || got.StopLoss != 75400 || got.TakeProfit != 76800 {
		t.Fatalf("bad prices: %+v", got)
	}
	if got.Lot != 0.01 {
		t.Fatalf("bad lot: %+v", got)
	}
}

func TestExtractDecision_NoFence(t *testing.T) {
	reply := "Chưa đủ confluence, đợi M15 đóng nến xác nhận trên EMA20. Đang chờ."
	if got := ExtractDecision(reply); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestExtractDecision_InvalidAction(t *testing.T) {
	reply := "```json\n" +
		`{"action":"MAYBE","symbol":"BTCUSDT","entry":1,"stop_loss":1,"take_profit":1,"lot":1}` + "\n```"
	if got := ExtractDecision(reply); got != nil {
		t.Fatalf("expected nil for bad action, got %+v", got)
	}
}

func TestExtractDecision_MissingField(t *testing.T) {
	reply := "```json\n" +
		`{"action":"BUY","symbol":"BTCUSDT","entry":100}` + "\n```"
	if got := ExtractDecision(reply); got != nil {
		t.Fatalf("expected nil for missing sl/tp, got %+v", got)
	}
}

func TestExtractDecision_MissingLot(t *testing.T) {
	reply := "```json\n" +
		`{"action":"BUY","symbol":"BTCUSDT","entry":100,"stop_loss":99,"take_profit":101}` + "\n```"
	if got := ExtractDecision(reply); got != nil {
		t.Fatalf("expected nil for missing lot, got %+v", got)
	}
}

func TestFormatAdvisorReplyForUser_SizedByRisk(t *testing.T) {
	// Defaults: ADVISOR_ACCOUNT_USDT=1000, ADVISOR_RISK_PCT=0.5 ⇒ risk $5.
	// XAUUSDT: 1 lot = 100 oz. Delta(entry,SL)=|4760-4770|=10.
	//   sized lot = 5 / (10 * 100) = 0.005
	//   TP PnL   = |4760-4740| * 0.005 * 100 = +10.00  (+1.00% of $1000)
	//   SL PnL   = -10 * 0.005 * 100         =  -5.00  (-0.50% of $1000)
	//   R:R      = 10 / 5                    = 2.00
	t.Setenv("ADVISOR_ACCOUNT_USDT", "")
	t.Setenv("ADVISOR_RISK_PCT", "")

	raw := "Vào SELL.\n\n```json\n" +
		`{"action":"SELL","symbol":"XAUUSDT","entry":4760,"stop_loss":4770,"take_profit":4740,"lot":0.1}` + "\n```"
	d := ExtractDecision(raw)
	if d == nil {
		t.Fatal("expected decision")
	}
	out := FormatAdvisorReplyForUser(raw, d)

	if !strings.Contains(out, "XAUUSDT") || !strings.Contains(out, "SELL") {
		t.Fatalf("missing symbol/action: %s", out)
	}
	if !strings.Contains(out, "Vốn $1000") {
		t.Fatalf("missing account header: %s", out)
	}
	if !strings.Contains(out, "• Khối lượng (base): 0.005") {
		t.Fatalf("lot not resized to 0.005: %s", out)
	}
	if !strings.Contains(out, "• TP: +10.00 USDT (+1.00%)") {
		t.Fatalf("unexpected TP line: %s", out)
	}
	if !strings.Contains(out, "• SL: -5.00 USDT (-0.50%)") {
		t.Fatalf("unexpected SL line: %s", out)
	}
	if !strings.Contains(out, "R:R 2.00") {
		t.Fatalf("missing R:R: %s", out)
	}
}

func TestFormatAdvisorReplyForUser_LegacyWhenAccountZero(t *testing.T) {
	// ADVISOR_ACCOUNT_USDT=0 disables sizing: we keep the LLM's raw lot
	// and show absolute-USDT PnL (pre-risk-sizing behaviour).
	t.Setenv("ADVISOR_ACCOUNT_USDT", "0")
	t.Setenv("ADVISOR_RISK_PCT", "0.5")

	raw := "Vào SELL.\n\n```json\n" +
		`{"action":"SELL","symbol":"XAUUSDT","entry":4760,"stop_loss":4770,"take_profit":4740,"lot":0.1}` + "\n```"
	d := ExtractDecision(raw)
	if d == nil {
		t.Fatal("expected decision")
	}
	out := FormatAdvisorReplyForUser(raw, d)
	if !strings.Contains(out, "Ước tính PnL") {
		t.Fatalf("expected legacy PnL section: %s", out)
	}
	if !strings.Contains(out, "• Nếu chạm TP: +200.00 USDT") {
		t.Fatalf("unexpected TP PnL (raw lot=0.1 * 20 * 100 = 200): %s", out)
	}
	if !strings.Contains(out, "• Nếu chạm SL: -100.00 USDT") {
		t.Fatalf("unexpected SL PnL (raw lot=0.1 * -10 * 100 = -100): %s", out)
	}
}

func TestStripDecisionFence(t *testing.T) {
	prose := "Vào BUY theo setup M15 pin bar."
	reply := prose + "\n\n```json\n" +
		`{"action":"BUY","symbol":"BTCUSDT","entry":1,"stop_loss":1,"take_profit":1,"lot":0.5}` + "\n```"
	got := StripDecisionFence(reply)
	if got != prose {
		t.Fatalf("expected %q, got %q", prose, got)
	}
}

func TestExtractDecision_LowercaseNormalises(t *testing.T) {
	reply := "```json\n" +
		`{"action":"sell ","symbol":" btcusdt ","entry":75000,"stop_loss":75500,"take_profit":74000,"lot":0.02}` + "\n```"
	got := ExtractDecision(reply)
	if got == nil {
		t.Fatalf("expected payload")
	}
	if got.Action != "SELL" || got.Symbol != "BTCUSDT" {
		t.Fatalf("bad normalisation: %+v", got)
	}
}
