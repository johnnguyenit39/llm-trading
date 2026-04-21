package model

import "time"

// Turn is one message in a conversation: either from the user or from the
// assistant. Role values match the OpenAI/DeepSeek chat-completion schema
// so we can pass Turns straight through to the LLM.
type Turn struct {
	Role    string    `json:"role"`    // "user" | "assistant" | "system"
	Content string    `json:"content"` // raw text
	Time    time.Time `json:"time"`
}

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)
