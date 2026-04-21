package biz

import (
	"context"

	"j_ai_trade/modules/advisor/model"
)

// LLMProvider is the abstraction over any streaming chat-completion backend.
// Phase 1 ships a DeepSeek implementation; swapping in OpenAI, Anthropic,
// Gemini, a local Ollama, etc. only requires a new struct that satisfies
// this interface — no changes to ChatHandler or the Telegram adapter.
//
// Contract:
//   - Stream MUST eventually close both returned channels.
//   - At most one value is delivered on the error channel.
//   - Partial output before an error is valid and should not be discarded.
//   - Implementations are responsible for auth, retries, and SSE parsing;
//     they receive a canonical []model.Turn (role/content) and emit raw
//     content deltas on the chunks channel.
type LLMProvider interface {
	Stream(ctx context.Context, turns []model.Turn) (<-chan string, <-chan error)

	// Name returns an identifier used only for logging/metrics (e.g.
	// "deepseek:deepseek-chat", "openai:gpt-4o-mini"). Never user-facing.
	Name() string
}
