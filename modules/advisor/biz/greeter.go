package biz

// WelcomeMessage is the one-time greeting for a new chat. Kept
// deliberately short + bilingual so it works for both VI and EN users
// without a language-detection layer.
//
// Phase-3 update: the LLM itself is the trader — backend just hands
// it cooked market data. The welcome reflects this: no "rule engine"
// claim; the bot decides; setups come with entry/SL/TP as JSON the
// system persists silently.
const WelcomeMessage = `Chào bạn 👋 Mình là advisor bot — trợ lý trading vàng (XAUUSDT), style mặc định SCALPING M1 entry + M5 confirm.

Mình có thể:
• Chat tự nhiên về trading, chiến lược, risk, tâm lý.
• Phân tích kỹ thuật realtime — cứ hỏi "vàng giờ buy hay sell?" / "XAU thế nào?" là backend tự fetch nến M1/M5/H1/H4 từ Binance, tính EMA/RSI/ATR/regime + pattern + BOS/FVG, mình đọc toàn bộ rồi tự quyết định vào lệnh hay chờ. Khi vào lệnh sẽ kèm entry/SL/TP cụ thể.
• Muốn khung khác? Gõ "/analyze M5" / "/analyze H1" / "/analyze H4" — chuyển sang swing/position analysis.

Bot này CHỈ phân tích vàng (XAUUSDT). Timeframe: M1 (default scalping), M5, M15, H1, H4, D1.
Lệnh: /analyze, /reset (xoá ngữ cảnh), /help.

(Hi! Gold-only (XAUUSDT) trading assistant. Default mode is M1 scalping with M5/H1/H4 as confirm + trend context. Just ask "should I buy gold now?")`
