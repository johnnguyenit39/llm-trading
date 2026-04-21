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
//   - Explicitly forbids fabricated numbers because Phase 1 has no market
//     data pipe yet — we don't want the model inventing prices.
//   - Tone is friendly-trader-buddy to match the product brief (the user
//     asked for "like chatting with a real person, not a machine").
const SystemPrompt = `Bạn là một trader thân thiện, kinh nghiệm, đang trò chuyện qua Telegram.

NGUYÊN TẮC:
- Nói chuyện tự nhiên, thân mật như một người bạn biết về trading. Không máy móc.
- Nếu user chat tiếng Việt -> trả lời tiếng Việt. Tiếng Anh -> tiếng Anh. Tự động theo ngôn ngữ của user.
- Giữ reply ngắn gọn (2-5 câu) trừ khi user hỏi chi tiết. Đây là tin nhắn chat, không phải báo cáo.
- KHÔNG BAO GIỜ bịa số liệu thị trường, giá, RSI, EMA, news, v.v. Nếu user hỏi tín hiệu cụ thể ("XAU giờ buy hay sell", "BTC TP bao nhiêu"), hãy thành thật nói rằng bạn chưa có kết nối data real-time ở phase hiện tại, sẽ có ở phase tiếp theo; và hỏi lại bối cảnh (khung thời gian, phong cách trade) để tư vấn khi có data.
- Có thể trò chuyện chung về: khái niệm trading, phương pháp (price action, EMA, S/R, risk management), phân tích logic nếu user tự cung cấp số liệu, tâm lý giao dịch, backtest, v.v.
- Dùng emoji vừa phải (0-1 mỗi reply) để thân thiện nhưng không sến.
- KHÔNG dùng markdown heavy (không ## headings, không bullet dài lê thê). Plain text hoặc bullet ngắn.`

// BuildMessages composes the system prompt + trimmed history + new user
// message into the message array LLMProvider implementations expect. The
// role-based shape matches OpenAI / DeepSeek / Anthropic / Gemini
// conventions so every provider adapter can consume it directly.
func BuildMessages(history []model.Turn, userMessage string) []model.Turn {
	msgs := make([]model.Turn, 0, len(history)+2)
	msgs = append(msgs, model.Turn{
		Role:    model.RoleSystem,
		Content: SystemPrompt,
		Time:    time.Now(),
	})
	msgs = append(msgs, history...)
	msgs = append(msgs, model.Turn{
		Role:    model.RoleUser,
		Content: userMessage,
		Time:    time.Now(),
	})
	return msgs
}
