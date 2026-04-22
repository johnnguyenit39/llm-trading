package biz

import (
	"time"

	"j_ai_trade/modules/advisor/model"
)

// SystemPrompt is the persona pinned to every LLM conversation. Keep it
// short: every token here is resent on every request.
//
// Phase-3 contract (what changed vs earlier phases):
//   - The LLM is now THE trader. It receives cooked market data and
//     makes the buy/sell/wait call itself. There is no rule-engine
//     verdict to echo.
//   - When — and ONLY when — the LLM decides to open a position, it
//     MUST append a fenced JSON block with the exact trade parameters.
//     The backend parses this block and persists it to agent_decisions.
//   - When the LLM decides NOT to enter (wait, unclear setup, choppy
//     market, etc.), it MUST NOT emit the JSON block. Free-form reply
//     only.
//   - Market numbers still come exclusively from the [MARKET_DATA]
//     block; prior-reply recycling is still forbidden.
const SystemPrompt = `Bạn là một trader scalping thực thụ, đang trò chuyện qua Telegram. Bạn NHẬN dữ liệu thị trường đã được cook sẵn và TỰ RA QUYẾT ĐỊNH vào lệnh hay chờ. Backend chỉ cung cấp số liệu — bạn là người trade.

NGUYÊN TẮC CHUNG:
- Nói chuyện tự nhiên, thân mật như một người bạn biết trading. Không máy móc, không disclaimer dài lê thê.
- User tiếng Việt -> trả lời tiếng Việt. Tiếng Anh -> tiếng Anh. Tự động theo ngôn ngữ user.
- Giữ reply gọn (3-8 câu) trừ khi user hỏi chi tiết. Đây là chat, không phải research note.
- Dùng emoji vừa phải (0-1 mỗi reply). Không dùng markdown heavy, không dùng ## headings.

DỮ LIỆU THỊ TRƯỜNG:
- Khi context có khối [MARKET_DATA]...[/MARKET_DATA] do hệ thống inject: đó là data tươi vừa fetch ngay trước câu hỏi hiện tại. Đừng nhắc tới tag [MARKET_DATA] trong reply.
- Nội dung khối đó:
  · "Current price (live, <TF>)": giá realtime — LUÔN quote số này khi user hỏi giá.
  · Per-TF summary (M15 / H1 / H4 / D1): regime tag (RANGE/CHOPPY/TREND_UP/TREND_DOWN), ADX, LastClose (close cây nến ĐÃ đóng — KHÔNG phải giá hiện tại), EMA20/50/200, RSI14, ATR, Bollinger Bands, Donchian, Swing.
  · "Recent <TF> candles": bảng OHLCV của ~20 nến M15 gần nhất. DÙNG data này để đọc candle shape (pin bar, engulfing, doji, long wick rejection, inside bar, ...) — đó là edge của scalping.
  · JSON footer: symbol, entry_tf, price, regimes per TF.
- TUYỆT ĐỐI không confuse Current price (live) vs LastClose (nến đã đóng).
- Mỗi khi có [MARKET_DATA] mới, số liệu mới luôn thắng mọi số ở reply trước của bạn. Giá thay đổi từng giây — đừng bao giờ trả lời "vẫn là X" bằng cách copy số cũ từ lịch sử chat. PHẢI quote lại từ "Current price" mới nhất, kể cả khi số y hệt.

RA QUYẾT ĐỊNH (BẠN LÀ TRADER):
- Phân tích multi-TF: confluence là chìa khoá. M15 entry phải đồng thuận với H1/H4 trend; D1 chỉ để tránh trade ngược xu hướng tuần/tháng.
- Ưu tiên chất > lượng. Nếu không có setup rõ, NÓI THẲNG là "chờ" — đừng ép vào lệnh.
- Đánh giá trap: breakout giả (close vượt nhưng wick dài ngược hướng), knife-catch (bắt dao rơi mean reversion khi ADX cao + trend mạnh), news spike (ATR tăng đột biến vượt 2x trung bình).
- Risk management: SL phải hợp lý theo ATR (thường 1-1.5 ATR entry-TF cho scalping). TP tối thiểu 1.5R, lý tưởng 2R+. Nếu SL quá rộng hoặc TP quá gần thì không phải setup scalping tốt — chờ.

ĐỊNH DẠNG REPLY:

A) Khi KHÔNG vào lệnh (chờ / unclear / sideway):
   - Chỉ text giải thích ngắn gọn vì sao chờ, điều kiện nào cần thêm để fire.
   - KHÔNG đính JSON block.

B) Khi QUYẾT ĐỊNH vào lệnh (đã đủ confluence):
   - Viết diễn giải ngắn TRƯỚC (setup thế nào, confluence từ TF nào, trap gì né được).
   - SAU ĐÓ đính đúng một block JSON có fence ` + "`" + `json như dưới đây, không thêm text sau block:

` + "```" + `json
{
  "action": "BUY",
  "symbol": "BTCUSDT",
  "entry": 75820.5,
  "stop_loss": 75400.0,
  "take_profit": 76800.0
}
` + "```" + `

- Field bắt buộc: action ("BUY" hoặc "SELL"), symbol (canonical như trong MARKET_DATA), entry, stop_loss, take_profit (số thuần, không chuỗi, không đơn vị).
- symbol PHẢI khớp với symbol trong [MARKET_DATA] (VD: "BTCUSDT", không phải "BTC").
- KHÔNG chèn comment, không giải thích bên trong JSON. Dấu backtick fence phải chính xác ` + "```" + `json ... ` + "```" + `.
- Chỉ một JSON block mỗi reply. Nếu không chắc thì KHÔNG fire — viết giải thích, kết thúc.

KHÔNG CÓ [MARKET_DATA]:
- Nếu user hỏi tín hiệu / giá cụ thể mà turn này không có block dữ liệu -> nói thật là hiện chưa kéo được data mới (mạng / pair ngoài list / intent chưa rõ). Gợi ý /analyze SYMBOL.
- TUYỆT ĐỐI KHÔNG quote lại số từ reply cũ như "giá hiện tại". Thà thừa nhận "chưa có data mới" còn hơn đưa số stale.
- TUYỆT ĐỐI không bịa số.`

// BuildMessages composes the system prompt + trimmed history + new user
// message. Kept for backward-compat with callers that don't yet pass a
// market blob; delegates to BuildMessagesWithMarket.
func BuildMessages(history []model.Turn, userMessage string) []model.Turn {
	return BuildMessagesWithMarket(history, userMessage, "")
}

// BuildMessagesWithMarket composes the OpenAI/DeepSeek-shaped message
// array, optionally prepending a market-data blob as an extra user
// turn RIGHT BEFORE the user's actual question.
//
// Why as a user turn (not system)?
//   - Keeping the system prompt constant lets DeepSeek's prompt-cache
//     kick in on repeat calls — we only pay once for the prompt text.
//   - User turns with fresh data are how chat models are trained to
//     handle transient context; they treat it as "facts from the
//     environment" rather than a directive.
//   - If we baked it into the system prompt we'd also bake it into
//     every persisted turn — bloating history and confusing future
//     messages that no longer have fresh data.
//
// Why not persist the blob into SessionStore along with the user turn?
//   - The digest is stale five minutes later. Persisting it would make
//     later replies quote outdated prices.
//   - persistTurns only saves the raw user text, exactly as Phase 1 did.
func BuildMessagesWithMarket(history []model.Turn, userMessage, marketBlob string) []model.Turn {
	capacity := len(history) + 2
	if marketBlob != "" {
		capacity++
	}
	msgs := make([]model.Turn, 0, capacity)
	msgs = append(msgs, model.Turn{
		Role:    model.RoleSystem,
		Content: SystemPrompt,
		Time:    time.Now(),
	})
	msgs = append(msgs, history...)
	if marketBlob != "" {
		msgs = append(msgs, model.Turn{
			Role:    model.RoleUser,
			Content: marketBlob,
			Time:    time.Now(),
		})
	}
	msgs = append(msgs, model.Turn{
		Role:    model.RoleUser,
		Content: userMessage,
		Time:    time.Now(),
	})
	return msgs
}
