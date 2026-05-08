// Package provider exposes a single New() factory that reads AI_PROVIDER
// from env and returns the matching biz.LLMProvider. Callers never import
// a concrete provider package directly — only this factory and the
// biz.LLMProvider interface cross the package boundary.
//
// Supported values for AI_PROVIDER:
//
//	"deep-seek"  (default when unset) — uses DEEP_SEEK_API_KEY
//	"claude"                          — uses CLAUDE_API_KEY
package provider

import (
	"fmt"
	"os"

	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/provider/claude"
	"j_ai_trade/modules/advisor/provider/deepseek"
)

// New returns the LLMProvider selected by the AI_PROVIDER env var.
// Missing or empty AI_PROVIDER defaults to "deep-seek".
func New() (biz.LLMProvider, error) {
	return NewConfigured(nil, nil)
}

// NewConfigured is like New but allows pinning temperature and seed for
// reproducible runs. Both parameters are optional (pass nil for defaults).
// Seed is forwarded to DeepSeek only; Anthropic does not expose a seed
// parameter so it is silently ignored for the claude provider.
func NewConfigured(temperature *float64, seed *int) (biz.LLMProvider, error) {
	p := os.Getenv("AI_PROVIDER")
	switch p {
	case "", "deep-seek":
		c, err := deepseek.New()
		if err != nil {
			return nil, err
		}
		if temperature != nil {
			c = c.WithTemperature(*temperature)
		}
		if seed != nil {
			c = c.WithSeed(*seed)
		}
		return c, nil

	case "claude":
		c, err := claude.New()
		if err != nil {
			return nil, err
		}
		if temperature != nil {
			c = c.WithTemperature(*temperature)
		}
		return c, nil

	default:
		return nil, fmt.Errorf("unknown AI_PROVIDER %q (supported: deep-seek, claude)", p)
	}
}
