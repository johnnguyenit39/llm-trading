package biz

import (
	"encoding/json"
	"regexp"
	"strings"
)

// DecisionPayload is the shape of the JSON block the LLM emits when it
// decides to open a trade. Fields match the Postgres schema 1:1 so the
// handler can map straight from JSON into the ORM model without a
// second DTO layer.
//
// Numeric fields MUST be JSON numbers (not strings) — the system
// prompt explicitly tells the LLM this. We still parse with
// float64 target so that Python-style "75820.5" numerics work.
type DecisionPayload struct {
	Action     string  `json:"action"`
	Symbol     string  `json:"symbol"`
	Entry      float64 `json:"entry"`
	StopLoss   float64 `json:"stop_loss"`
	TakeProfit float64 `json:"take_profit"`
}

// decisionFenceRe matches a ```json ... ``` fenced block with any
// amount of whitespace inside. The `(?s)` flag makes `.` match
// newlines. We deliberately anchor on ```json (lowercase) only —
// uppercase ```JSON or bare ``` wouldn't be reliable to detect as an
// intentional decision block, and the system prompt pins the exact
// lowercase form.
var decisionFenceRe = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")

// ExtractDecision scans an LLM reply for the ```json {...} ``` fenced
// decision block defined in SystemPrompt. Returns the parsed payload
// when found and structurally valid; nil when the reply contains no
// block (the normal "explain only, don't trade" path) or when the
// payload fails validation.
//
// Non-fatal by design: if the LLM emits a malformed block we'd rather
// skip persistence than crash the reply path. The caller logs and
// moves on. Users will still see the prose reply; worst case is a
// missed DB row that can be manually recovered from the chat log.
func ExtractDecision(reply string) *DecisionPayload {
	match := decisionFenceRe.FindStringSubmatch(reply)
	if len(match) < 2 {
		return nil
	}
	var p DecisionPayload
	if err := json.Unmarshal([]byte(match[1]), &p); err != nil {
		return nil
	}
	if !p.valid() {
		return nil
	}
	p.normalise()
	return &p
}

// StripDecisionFence removes the ```json ... ``` block (and one
// trailing newline) from the LLM reply so the persisted chat history
// and the user-visible bubble stay clean prose. We keep the JSON out
// of session history because (a) it's noise for downstream turns and
// (b) it can confuse the LLM into thinking "I already traded this"
// when the user asks a follow-up.
func StripDecisionFence(reply string) string {
	cleaned := decisionFenceRe.ReplaceAllString(reply, "")
	return strings.TrimSpace(cleaned)
}

// valid enforces the prompt contract: non-empty symbol, a recognised
// action, and three positive prices. We don't check price ORDERING
// (SL<Entry for BUY vs SL>Entry for SELL) here — that's a business
// rule worth logging but not worth refusing to persist over, since a
// malformed-looking row is useful debug evidence.
func (p DecisionPayload) valid() bool {
	if p.Symbol == "" {
		return false
	}
	act := strings.ToUpper(strings.TrimSpace(p.Action))
	if act != "BUY" && act != "SELL" {
		return false
	}
	if p.Entry <= 0 || p.StopLoss <= 0 || p.TakeProfit <= 0 {
		return false
	}
	return true
}

// normalise canonicalises the free-form parts of the payload so the
// storage layer sees consistent data (upper-case symbol+action,
// whitespace stripped). Runs AFTER valid() so we don't mutate a
// payload we're about to reject.
func (p *DecisionPayload) normalise() {
	p.Symbol = strings.ToUpper(strings.TrimSpace(p.Symbol))
	p.Action = strings.ToUpper(strings.TrimSpace(p.Action))
}
