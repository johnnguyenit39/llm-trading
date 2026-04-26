package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// AdvisorBot is a minimal Telegram Bot API client tailored for the advisor
// module. It deliberately does NOT reuse TelegramService because:
//   - The cron bot pushes to a fixed channel; the advisor bot replies to
//     arbitrary chat IDs.
//   - This client needs long-polling, editMessageText, and sendChatAction
//     which the legacy service does not implement.
//
// Bot token is read from env J_AI_TRADE_ADVISOR.
type AdvisorBot struct {
	token      string
	httpClient *http.Client
}

// NewAdvisorBot loads the token from env and returns a configured client.
// Returns error if the env var is empty — caller should treat as disabled.
func NewAdvisorBot() (*AdvisorBot, error) {
	token := os.Getenv("J_AI_TRADE_ADVISOR")
	if os.Getenv("ENV") != "DEV" {
		token = os.Getenv("J_AI_TRADE_ADVISOR_DEV")
	}

	if token == "" {
		return nil, errors.New("J_AI_TRADE_ADVISOR env var is empty")
	}
	return &AdvisorBot{
		token: token,
		// Long-poll uses its own timeout; this client-level timeout must be
		// larger than the longest long-poll (30s + buffer).
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (b *AdvisorBot) endpoint(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.token, method)
}

// GetUpdates performs a long-poll for new updates. `offset` is the next
// update_id to fetch (previous max + 1); `timeout` is server-side wait in
// seconds. Returns the updates (possibly empty) or an error.
func (b *AdvisorBot) GetUpdates(ctx context.Context, offset int64, timeout int) ([]Update, error) {
	params := url.Values{}
	params.Set("timeout", strconv.Itoa(timeout))
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	// Only subscribe to message updates for Phase 1.
	params.Set("allowed_updates", `["message"]`)

	reqURL := b.endpoint("getUpdates") + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram getUpdates returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed getUpdatesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("telegram getUpdates decode: %w", err)
	}
	if !parsed.OK {
		return nil, fmt.Errorf("telegram getUpdates ok=false")
	}
	return parsed.Result, nil
}

// SendMessage sends a plain-text message and returns the sent Message (for
// its message_id, needed for future edits).
func (b *AdvisorBot) SendMessage(ctx context.Context, chatID int64, text string) (*Message, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	var parsed sendMessageResponse
	if err := b.postJSON(ctx, "sendMessage", payload, &parsed); err != nil {
		return nil, err
	}
	return &parsed.Result, nil
}

// EditMessageText overwrites a previously sent message. Used by the stream
// editor to progressively reveal DeepSeek tokens.
func (b *AdvisorBot) EditMessageText(ctx context.Context, chatID, messageID int64, text string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	var parsed genericResponse
	if err := b.postJSON(ctx, "editMessageText", payload, &parsed); err != nil {
		return err
	}
	if !parsed.OK {
		// Telegram complains "message is not modified" when content is
		// identical. Treat as non-fatal to keep stream flow going.
		return fmt.Errorf("editMessageText: %s", parsed.Description)
	}
	return nil
}

// SendChatAction fires the "typing..." indicator. It expires after ~5s, so
// for long-running replies callers should tick this every 4s.
func (b *AdvisorBot) SendChatAction(ctx context.Context, chatID int64, action string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"action":  action,
	}
	var parsed genericResponse
	return b.postJSON(ctx, "sendChatAction", payload, &parsed)
}

func (b *AdvisorBot) postJSON(ctx context.Context, method string, payload any, out any) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint(method), bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram %s returned %d: %s", method, resp.StatusCode, string(body))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("telegram %s decode: %w", method, err)
		}
	}
	return nil
}
