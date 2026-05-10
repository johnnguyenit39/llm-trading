// Package claude is a concrete implementation of biz.LLMProvider backed
// by the Anthropic Messages API with SSE streaming.
//
// Wire format differs from OpenAI: system prompt is a top-level field,
// SSE events are typed (content_block_delta / message_stop), and auth
// uses x-api-key instead of Bearer. Everything else — the biz.LLMProvider
// interface and the Turn slice — is identical to the DeepSeek client.
package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/model"
)

// Anthropic Messages API wire types.

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// systemBlock is a single content block in the top-level system array.
// Setting CacheControl marks it as a prompt-cache candidate — Anthropic
// reuses the KV cache for this block on subsequent calls with identical text,
// cutting both latency and cost for the fixed system prompt.
type systemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

type apiRequest struct {
	Model     string        `json:"model"`
	Messages  []apiMessage  `json:"messages"`
	System    []systemBlock `json:"system,omitempty"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
	// Temperature is only emitted when explicitly set (omitempty + pointer).
	Temperature *float64 `json:"temperature,omitempty"`
}

// Anthropic SSE emits named events. We only need content_block_delta
// with type=text_delta; everything else (ping, message_start, etc.) is
// skipped.
type streamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// Client holds credentials and is the sole owner of the Anthropic API
// key. Swap providers by constructing a different biz.LLMProvider
// implementation — nothing outside this package needs to change.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client

	// temperature is nil unless the caller opts in via WithTemperature.
	temperature *float64
}

// New reads CLAUDE_API_KEY / CLAUDE_BASE_URL / CLAUDE_MODEL from env.
// Base URL and model default to Anthropic's public endpoint and
// claude-sonnet-4-6 respectively.
func New() (*Client, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("CLAUDE_API_KEY env var is empty")
	}

	baseURL := os.Getenv("CLAUDE_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	mdl := os.Getenv("CLAUDE_MODEL")
	if mdl == "" {
		mdl = "claude-sonnet-4-6"
	}

	maxTokens := 4096

	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      mdl,
		maxTokens:  maxTokens,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

// Name satisfies biz.LLMProvider. Format "claude:<model>" mirrors the
// DeepSeek convention and is stable for log grep.
func (c *Client) Name() string { return "claude:" + c.model }

// WithTemperature pins sampling temperature for all subsequent Stream
// calls. Backtest harnesses set 0 for near-deterministic output; live
// chat leaves it unset to use Anthropic's default (~1.0).
func (c *Client) WithTemperature(t float64) *Client {
	c.temperature = &t
	return c
}

// Stream satisfies biz.LLMProvider. It sends the canonical Turn slice to
// the Anthropic Messages endpoint and emits text deltas as they arrive.
// System turns are extracted and merged into the top-level system field;
// all other turns are sent as the messages array.
func (c *Client) Stream(ctx context.Context, turns []model.Turn) (<-chan string, <-chan error) {
	chunks := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errCh)

		var systemParts []string
		var msgs []apiMessage
		for _, t := range turns {
			switch t.Role {
			case model.RoleSystem:
				systemParts = append(systemParts, t.Content)
			default:
				msgs = append(msgs, apiMessage{Role: t.Role, Content: t.Content})
			}
		}

		reqBody, err := json.Marshal(apiRequest{
			Model:    c.model,
			Messages: msgs,
			System: []systemBlock{{
				Type:         "text",
				Text:         strings.Join(systemParts, "\n\n"),
				CacheControl: &cacheControl{Type: "ephemeral"},
			}},
			MaxTokens:   c.maxTokens,
			Stream:      true,
			Temperature: c.temperature,
		})
		if err != nil {
			errCh <- err
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			c.baseURL+"/v1/messages", bytes.NewReader(reqBody))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("anthropic returned %d: %s", resp.StatusCode, string(body))
			return
		}

		// Anthropic SSE format:
		//   event: content_block_delta
		//   data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"..."}}
		//
		// We track the current event type across lines and only decode
		// data payloads that follow a content_block_delta event header.
		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 256*1024)

		var currentEvent string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") {
				currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			// message_stop signals end of stream.
			if currentEvent == "message_stop" {
				return
			}
			if currentEvent != "content_block_delta" {
				continue
			}

			var ev streamEvent
			if err := json.Unmarshal([]byte(payload), &ev); err != nil {
				continue
			}
			if ev.Delta.Type != "text_delta" || ev.Delta.Text == "" {
				continue
			}
			select {
			case chunks <- ev.Delta.Text:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
			errCh <- err
		}
	}()

	return chunks, errCh
}

// Compile-time assertion that Client satisfies the interface.
var _ biz.LLMProvider = (*Client)(nil)
