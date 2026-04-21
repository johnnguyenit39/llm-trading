package ensembles

// DefaultSymbols is the canonical universe of symbols this system
// analyses, shared by the cron broadcaster and the advisor chat bot.
// Both consumers import the same slice so adding/removing a pair only
// requires one edit — otherwise the two modules would drift, and a
// signal fired on Telegram channel wouldn't match what the chat bot
// knows to discuss.
//
// All entries are Binance futures-style tickers. XAUUSDT is the
// tokenised-gold pair on Binance (not real forex XAU/USD).
var DefaultSymbols = []string{
	"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT",
	"XRPUSDT", "ADAUSDT", "AVAXUSDT", "LINKUSDT",
	"DOTUSDT", "ATOMUSDT", "NEARUSDT", "SUIUSDT",
	"DOGEUSDT", "TRXUSDT", "BCHUSDT", "LTCUSDT",
	"XAUUSDT",
}
