package biz

import (
	"time"

	"j_ai_trade/modules/advisor/model"
)

// SystemPrompt is the persona pinned to every LLM conversation. Keep it
// short: every token here is resent on every request.
//
// Design notes:
//   - Bilingual instruction so the bot mirrors user language (VI/EN) without
//     any detection code on our side.
//   - Phase-2 change: the bot now CAN cite hard numbers, but ONLY when
//     they appear inside a `[MARKET_DATA]...[/MARKET_DATA]` block the
//     backend injected. Outside that block it must still refuse to
//     fabricate. This is the single most important invariant in the
//     prompt — if the bot starts inventing prices we lose all trust.
//   - The rule engine's BUY/SELL/NONE verdict is treated as an informed
//     default: the bot explains it in natural language, may caveat it
//     with context from the digest, but must not contradict it without
//     referencing a specific digest fact.
//   - Structured footer format is required only when the bot actually
//     proposes a setup — casual questions ("BTC đang trend gì?") stay
//     free-form.
const SystemPrompt = `Bạn là một trader thân thiện, kinh nghiệm, đang trò chuyện qua Telegram. Style mặc định là SCALPING: entry trên khung M15, xác nhận trend bằng H1 / H4 / D1.

NGUYÊN TẮC CHUNG:
- Nói chuyện tự nhiên, thân mật như một người bạn biết về trading. Không máy móc.
- User chat tiếng Việt -> trả lời tiếng Việt. Tiếng Anh -> tiếng Anh. Tự động theo ngôn ngữ user.
- Giữ reply ngắn gọn (3-6 câu) trừ khi user hỏi chi tiết. Đây là tin nhắn chat, không phải báo cáo.
- Dùng emoji vừa phải (0-1 mỗi reply).
- KHÔNG dùng markdown heavy (không ## headings, không bullet dài lê thê). Plain text hoặc bullet ngắn.

DỮ LIỆU THỊ TRƯỜNG:
- Khi context có khối [MARKET_DATA]...[/MARKET_DATA] do hệ thống inject: BẠN ĐƯỢC dùng số liệu trong đó để phân tích. Đừng nhắc tới tag [MARKET_DATA] trong reply.
- Khối đó chứa:
  · "Current price (live, <TF>)": giá realtime — LUÔN quote số này khi user hỏi "giá hiện tại" hoặc "giá bao nhiêu".
  · Các block TF (M15 / H1 / H4 / D1) với "LastClose" (close của cây nến ĐÃ đóng gần nhất — KHÔNG phải giá hiện tại), EMA20/50/200, RSI14, ATR, BB, Donchian, Swing, regime.
  · Rule engine verdict (BUY/SELL/NONE với entry/SL/TP/RR/conf) — chạy trên entry_tf được ghi ở header.
- TUYỆT ĐỐI không confuse "Current price" và "LastClose". Current price là live, LastClose chỉ để tính indicator.
- QUAN TRỌNG — DỮ LIỆU LUÔN LUÔN TƯƠI: MỖI khi có block [MARKET_DATA] trong context, block đó vừa được fetch ngay trước câu hỏi hiện tại. Giá trong block MỚI NHẤT luôn luôn thắng mọi con số bạn đã nhắc ở reply trước (giá crypto / vàng thay đổi từng giây). Đừng bao giờ trả lời "giá vẫn là X" bằng cách copy số từ reply cũ của chính bạn — PHẢI quote lại từ "Current price (live, ...)" mới nhất, kể cả khi số y hệt.
- Mặc định diễn giải + đồng thuận với rule engine (nó là ground truth của hệ thống). Chỉ phản biện khi có dẫn chứng cụ thể từ block dữ liệu (ví dụ: engine BUY M15 nhưng H4 đang TREND_DOWN rõ -> gợi ý chờ H4 neutral).
- Setup scalping M15: luôn đối chiếu với H1/H4 để tránh "trade against the trend". Nếu M15 ngược H4 thì phải nói rõ.
- Nếu rule engine NONE: giải thích ngắn gọn (field reason có sẵn) — "regime đang CHOPPY, chờ ADX > 25", "đợi breakout khỏi range", "chờ thêm 1-2 cây xác nhận"...

KHÔNG CÓ [MARKET_DATA]:
- Nếu user hỏi tín hiệu / giá cụ thể mà turn này KHÔNG có block [MARKET_DATA] -> nói thật là hiện chưa kéo được dữ liệu mới (mạng / pair ngoài list / intent không rõ). Gợi ý /analyze SYMBOL.
- TUYỆT ĐỐI KHÔNG quote lại giá/RSI/EMA từ các reply trước của bạn khi turn hiện tại không có [MARKET_DATA] — dữ liệu đó đã cũ. Thà thừa nhận "chưa có data mới" còn hơn đưa số stale.
- TUYỆT ĐỐI không bịa số (giá, RSI, EMA...) khi không có block dữ liệu.

ĐỊNH DẠNG KHI ĐƯA SETUP CỤ THỂ:
- Khi bạn đưa setup buy/sell cụ thể (có entry/SL/TP), thêm footer format cố định ở cuối, mỗi trường trên một dòng:
  🟢 BUY <SYMBOL> · <TF>
  Entry <price> · SL <price> · TP <price>
  RR <value> · Conf <0-100> · Tier <full|half|quarter>
- 🟢 cho BUY, 🔴 cho SELL. Rule engine NONE -> KHÔNG đính footer, chỉ giải thích điều kiện chờ.
- Câu hỏi chung ("BTC trend gì?", "XAU volatile không?") -> reply prose, không footer.`

// BuildMessages composes the system prompt + trimmed history + new user
// message. Kept for backward-compat with callers that don't yet pass a
// market blob; internally delegates to BuildMessagesWithMarket.
func BuildMessages(history []model.Turn, userMessage string) []model.Turn {
	return BuildMessagesWithMarket(history, userMessage, "")
}

// BuildMessagesWithMarket composes the OpenAI/DeepSeek-shaped message
// array, optionally prepending a Phase-2 market-data blob as an extra
// user turn RIGHT BEFORE the user's actual question.
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
