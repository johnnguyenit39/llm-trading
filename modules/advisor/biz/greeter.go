package biz

// WelcomeMessage is the one-time greeting for a new chat. Kept
// deliberately short + bilingual so it works for both VI and EN users
// without a language-detection layer.
//
// Phase-2 update: now the bot CAN pull live data, so the message sells
// that capability and shows the exact command user can type. We keep
// the "just chat about trading" framing so new users don't feel
// obligated to formulate a structured request.
const WelcomeMessage = `Chào bạn 👋 Mình là advisor bot — trợ lý trading.

Mình có thể:
• Chat tự nhiên về trading, chiến lược, risk, tâm lý giao dịch.
• Phân tích kỹ thuật realtime khi bạn hỏi ví dụ "XAU giờ buy hay sell?" hoặc "BTC H4 thế nào?" — mình sẽ tự fetch candle, tính EMA/RSI/ATR/regime, chạy qua rule engine 4-chiến-thuật rồi giải thích.
• Lệnh: /analyze <SYMBOL> [TF] để ép phân tích (ví dụ /analyze BTC H4, /analyze XAU). /reset xoá ngữ cảnh. /help xem lệnh.

Pair đang support: BTC, ETH, SOL, XAU (gold), BNB, XRP, ADA, AVAX, LINK, DOT, ATOM, NEAR, SUI, DOGE, TRX, BCH, LTC. Timeframe: H1, H4, D1.

(Hi! I can chat about trading and run live technical analysis on the pairs above. Try /analyze BTC H4 or just ask "should I buy XAU?")`
