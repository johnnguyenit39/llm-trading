package biz

// WelcomeMessage is the one-time greeting for a new chat. Kept
// deliberately short + bilingual so it works for both VI and EN users
// without a language-detection layer.
//
// Phase-2 update: now the bot CAN pull live data, so the message sells
// that capability and shows the exact command user can type. We keep
// the "just chat about trading" framing so new users don't feel
// obligated to formulate a structured request.
const WelcomeMessage = `Chào bạn 👋 Mình là advisor bot — trợ lý trading, style mặc định SCALPING trên khung M15.

Mình có thể:
• Chat tự nhiên về trading, chiến lược, risk, tâm lý.
• Phân tích kỹ thuật realtime khi bạn hỏi "XAU giờ buy hay sell?" hay "BTC thế nào?" — mình tự fetch candle M15 + H1 + H4 + D1 từ Binance, tính EMA/RSI/ATR/regime, chạy qua rule engine 4-chiến-thuật (trend follow, mean reversion, breakout, structure) rồi giải thích setup kèm entry/SL/TP.
• Muốn khung khác? Gõ "/analyze BTC H4" hay "/analyze XAU D1" — mình chuyển sang swing/position analysis.

Pair đang support: BTC, ETH, SOL, XAU (gold), BNB, XRP, ADA, AVAX, LINK, DOT, ATOM, NEAR, SUI, DOGE, TRX, BCH, LTC.
Timeframe: M15 (default scalping), H1, H4, D1.
Lệnh: /analyze, /reset (xoá ngữ cảnh), /help.

(Hi! Default mode is M15 scalping with H1/H4/D1 as trend context. Try /analyze BTC or ask "should I buy XAU now?")`
