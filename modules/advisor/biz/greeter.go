package biz

// WelcomeMessage is the one-time greeting for a new chat. Kept
// deliberately short + bilingual so it works for both VI and EN users
// without a language-detection layer.
//
// Phase-3 update: the LLM itself is the trader — backend just hands
// it cooked market data. The welcome reflects this: no "rule engine"
// claim; the bot decides; setups come with entry/SL/TP as JSON the
// system persists silently.
const WelcomeMessage = `Chào bạn 👋 Mình là advisor bot — trợ lý trading AI, style mặc định SCALPING trên khung M15.

Mình có thể:
• Chat tự nhiên về trading, chiến lược, risk, tâm lý.
• Phân tích kỹ thuật realtime khi bạn hỏi "XAU giờ buy hay sell?" hay "BTC thế nào?" — backend tự fetch nến M15 + H1 + H4 + D1 từ Binance, tính EMA/RSI/ATR/regime + 20 nến OHLCV gần nhất, mình đọc toàn bộ rồi tự quyết định vào lệnh hay chờ. Khi vào lệnh sẽ kèm entry/SL/TP cụ thể.
• Muốn khung khác? Gõ "/analyze BTC H4" hay "/analyze XAU D1" — chuyển sang swing/position analysis.

Pair đang support: BTC, ETH, SOL, XAU (gold), BNB, XRP, ADA, AVAX, LINK, DOT, ATOM, NEAR, SUI, DOGE, TRX, BCH, LTC.
Timeframe: M15 (default scalping), H1, H4, D1.
Lệnh: /analyze, /reset (xoá ngữ cảnh), /help.

(Hi! Default mode is M15 scalping with H1/H4/D1 as trend context. Try /analyze BTC or ask "should I buy XAU now?")`
