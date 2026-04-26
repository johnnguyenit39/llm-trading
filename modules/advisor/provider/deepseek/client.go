// Package deepseek is a concrete implementation of biz.LLMProvider backed
// by the DeepSeek chat-completions API (OpenAI-compatible SSE streaming).
//
// This package is the ONLY place in the codebase that knows the DeepSeek
// wire format. To add another vendor (OpenAI, Anthropic, Gemini, Groq, a
// local Ollama), create a sibling package under provider/ that implements
// biz.LLMProvider — no other file changes.
package deepseek

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

// OpenAI-compatible chat-completion wire format. DeepSeek mirrors this 1:1.

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	// Temperature/Seed are only emitted when the caller explicitly opts
	// in (omitempty + pointer). Production chat leaves both nil so we
	// inherit DeepSeek's defaults (~0.7); the backtest harness sets
	// Temperature=0 + a fixed Seed for reproducibility.
	Temperature *float64 `json:"temperature,omitempty"`
	Seed        *int     `json:"seed,omitempty"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Client is the sole holder of the DeepSeek API key. It exposes a
// streaming interface so the ChatHandler can edit the reply bubble in
// place as tokens arrive.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client

	// temperature / seed are nil unless the caller explicitly opts in
	// via WithTemperature / WithSeed. Production leaves them nil so the
	// API uses its sensible defaults; the backtest harness pins both
	// for reproducible runs.
	temperature *float64
	seed        *int
}

// New reads DEEP_SEEK_API_KEY / DEEP_SEEK_BASE_URL / DEEP_SEEK_MODEL from
// env. Base URL and model have sensible defaults so minimal config is
// needed.
func New() (*Client, error) {
	apiKey := os.Getenv("DEEP_SEEK_API_KEY")
	if apiKey == "" {
		return nil, errors.New("DEEP_SEEK_API_KEY env var is empty")
	}

	baseURL := os.Getenv("DEEP_SEEK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	mdl := os.Getenv("DEEP_SEEK_MODEL")
	if mdl == "" {
		mdl = "deepseek-chat"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   mdl,
		// deepseek-reasoner can take 10s+ to first token; streaming keeps
		// the connection alive, but we still want a guard for stuck
		// requests. 120s is generous.
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

// Name satisfies biz.LLMProvider. Format "<vendor>:<model>" is stable and
// cheap to grep for in logs.
func (c *Client) Name() string { return "deepseek:" + c.model }

// WithTemperature pins the sampling temperature for every subsequent
// Stream call. Backtest pipelines set 0 to make the model effectively
// deterministic (still ~99% reproducible — minor floating-point drift
// across GPU batches remains). Live chat leaves it unset.
func (c *Client) WithTemperature(t float64) *Client {
	c.temperature = &t
	return c
}

// WithSeed pins the request seed. Combined with temperature=0 this
// gives byte-identical responses for byte-identical messages. Backtest
// only.
func (c *Client) WithSeed(s int) *Client {
	c.seed = &s
	return c
}

// Stream satisfies biz.LLMProvider. It sends the canonical Turn slice to
// DeepSeek and emits content deltas as they arrive. The chunk channel
// closes when the response body ends, ctx is cancelled, or an error
// occurs. At most one value is sent on errCh.
func (c *Client) Stream(ctx context.Context, turns []model.Turn) (<-chan string, <-chan error) {
	chunks := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errCh)

		msgs := make([]chatMessage, 0, len(turns))
		for _, t := range turns {
			msgs = append(msgs, chatMessage{Role: t.Role, Content: t.Content})
		}

		reqBody, err := json.Marshal(chatRequest{
			Model:       c.model,
			Messages:    msgs,
			Stream:      true,
			Temperature: c.temperature,
			Seed:        c.seed,
		})
		if err != nil {
			errCh <- err
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			c.baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("deepseek returned %d: %s", resp.StatusCode, string(body))
			return
		}

		// SSE wire format: each event is "data: {json}\n\n"; the sentinel
		// "data: [DONE]" marks end of stream. bufio.Scanner's default 64KB
		// line cap is plenty for a single delta line.
		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 256*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				return
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				// Malformed delta: skip silently. Some providers emit
				// keepalive comments we don't care about.
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			delta := chunk.Choices[0].Delta.Content
			if delta == "" {
				continue
			}
			select {
			case chunks <- delta:
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
