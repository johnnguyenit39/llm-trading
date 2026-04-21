package telegram

// Update represents a single incoming update from Telegram getUpdates.
// Only the fields Phase-1 needs are modeled; Telegram returns many more we
// simply ignore.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
}

type getUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type sendMessageResponse struct {
	OK     bool    `json:"ok"`
	Result Message `json:"result"`
}

type genericResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}
