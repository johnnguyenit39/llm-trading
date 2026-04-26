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

// marketDataBlockRe matches the fenced [MARKET_DATA]...[/MARKET_DATA]
// region, case-insensitive. `(?is)` = case-insensitive + dot matches
// newlines. Non-greedy so multiple blocks in one reply each get
// excised cleanly.
var marketDataBlockRe = regexp.MustCompile(`(?is)\[MARKET_DATA\].*?\[/MARKET_DATA\]`)

// marketDataStartTagRe catches a stray opening tag on its own line
// when the model echoed the header without a closing tag.
var marketDataStartTagRe = regexp.MustCompile(`(?im)^\s*\[/?MARKET_DATA\]\s*$`)

// marketDataLineRe matches individual "key: value" dump lines the LLM
// copies out of the market blob. Anchored at start-of-line only —
// prose that mentions these words mid-sentence is untouched. The key
// list covers every label emitted by market/digest plus the TF headers
// "M15:" / "H1:" / "H4:" / "D1:" on their own line.
var marketDataLineRe = regexp.MustCompile(
	`(?im)^\s*(` +
		`symbol|current_price|tf_alignment|session|prev_day|news|` +
		`regime|adx|stack|lastclose|` +
		`ema\d+|rsi\d*|atr|bbwidth|bb|donchian|swing|` +
		`close|nearestr|nearests|mom\d+|structure|vol|` +
		`rsi_div|ema_cross_bull_\d*|ema_cross_bear_\d*|bb_squeeze_releasing|` +
		`[MH]\d+` +
		`)\s*:.*$`,
)

// recentOrLastBlockRe kills the "Recent <TF> pivots / candles" and
// "Last N <TF> bar patterns" headers plus their payload — the rows
// underneath are formatted with prefixes like "[-1] 2026-04-22 14:30"
// or "SH 2386.5 14:00 LH" that we also strip below.
var recentOrLastBlockRe = regexp.MustCompile(
	`(?im)^\s*(recent (m1|m5|m15|h1|h4|d1)|last \d+ (m1|m5|m15|h1|h4|d1) bar patterns).*$`,
)

// pivotOrPatternRowRe matches the single-row outputs beneath those
// headers: pivot rows ("SH 2386.5 14:00 LH") and the "[-k] date time
// kind · r=X ..." pattern rows.
var pivotOrPatternRowRe = regexp.MustCompile(
	`(?im)^\s*(\[-?\d+\]|SH|SL)\s.*$`,
)

// multipleBlankLinesRe collapses 3+ newlines to 2 after stripping,
// so the remaining prose doesn't have big gaps.
var multipleBlankLinesRe = regexp.MustCompile(`\n{3,}`)

// StripMarketDataDump removes any chunk of the [MARKET_DATA] context
// block the LLM echoed back into its reply. It's a safety net: the
// SystemPrompt already forbids echoing, but DeepSeek occasionally
// disobeys and the user reads these replies on Telegram — a leaked
// dump is jarring and useless to them.
//
// Strategy, in order:
//  1. Remove full [MARKET_DATA]...[/MARKET_DATA] blocks (bracketed case).
//  2. Remove stray opening/closing tags on their own line.
//  3. Remove "key: value" lines whose key matches the digest vocab
//     (includes "news" for echoed "News: ..." lines).
//  4. Remove "Recent <TF> ..." / "Last N <TF> bar patterns" headers
//     AND the rows underneath (pivot or pattern).
//  5. Collapse 3+ blank lines to a single blank line for readability.
//
// We deliberately DO NOT touch prose that only mentions numbers in
// passing (e.g. "giá 2368 đang ở trên EMA20") — the line-anchored
// regex only fires on "EMA20: 2366" style rows.
func StripMarketDataDump(s string) string {
	s = marketDataBlockRe.ReplaceAllString(s, "")
	s = marketDataStartTagRe.ReplaceAllString(s, "")
	s = recentOrLastBlockRe.ReplaceAllString(s, "")
	s = pivotOrPatternRowRe.ReplaceAllString(s, "")
	s = marketDataLineRe.ReplaceAllString(s, "")
	s = multipleBlankLinesRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
