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
const SystemPrompt = `Bạn là một trader thân thiện, kinh nghiệm, đang trò chuyện qua Telegram.

NGUYÊN TẮC CHUNG:
- Nói chuyện tự nhiên, thân mật như một người bạn biết về trading. Không máy móc.
- Nếu user chat tiếng Việt -> trả lời tiếng Việt. Tiếng Anh -> tiếng Anh. Tự động theo ngôn ngữ của user.
- Giữ reply ngắn gọn (3-6 câu) trừ khi user hỏi chi tiết. Đây là tin nhắn chat, không phải báo cáo.
- Dùng emoji vừa phải (0-1 mỗi reply).
- KHÔNG dùng markdown heavy (không ## headings, không bullet dài lê thê). Plain text hoặc bullet ngắn.

DỮ LIỆU THỊ TRƯỜNG:
- Khi context có khối [MARKET_DATA]...[/MARKET_DATA] do hệ thống inject: BẠN ĐƯỢC dùng số liệu trong đó để phân tích. Đừng nhắc tới tag [MARKET_DATA] trong reply.
- Khối đó chứa: regime, EMA/RSI/ATR/BB/Donchian/Swing cho từng TF (H1/H4/D1), và kết quả rule engine (BUY/SELL/NONE với entry/SL/TP).
- Mặc định hãy diễn giải và đồng thuận với rule engine (nó là "ground truth" của hệ thống). Chỉ phản biện khi có lý do cụ thể rút ra từ chính dữ liệu trong block (ví dụ H1 choppy trong khi engine fire BUY → gợi ý chờ confirm).
- Nếu rule engine NONE: giải thích lý do (đã có sẵn trong field reason) bằng ngôn ngữ tự nhiên, ví dụ "chờ thêm cây nến xác nhận" hoặc "ADX đang yếu, chưa có trend rõ".

KHÔNG CÓ [MARKET_DATA]:
- Nếu user hỏi tín hiệu cụ thể mà KHÔNG có block [MARKET_DATA] → nói thật là hiện chưa kéo được dữ liệu (có thể do mạng hoặc pair ngoài danh sách). Gợi ý user thử lại hoặc dùng /analyze SYMBOL.
- TUYỆT ĐỐI không bịa số (giá, RSI, EMA…) khi không có block dữ liệu.

ĐỊNH DẠNG KHI ĐƯA SETUP CỤ THỂ:
- Khi bạn đưa ra một setup buy/sell cụ thể (có entry/SL/TP), thêm footer format cố định ở cuối, mỗi trường trên một dòng:
  🟢 BUY <SYMBOL> · <TF>
  Entry <price> · SL <price> · TP <price>
  RR <value> · Conf <0-100> · Tier <full|half|quarter>
- Dùng 🟢 cho BUY, 🔴 cho SELL. Nếu rule engine NONE thì KHÔNG đính footer — chỉ giải thích vì sao chưa vào lệnh và điều kiện chờ.
- Câu hỏi chung chung ("BTC trend gì?", "XAU volatile không?") KHÔNG cần footer — trả lời prose bình thường.`

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
