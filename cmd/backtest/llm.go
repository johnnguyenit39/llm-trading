package main

import (
	"context"
	"strings"

	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/model"
	"j_ai_trade/modules/advisor/provider/deepseek"
)

// llmRunner wraps the streaming DeepSeek client + cache so the backtest
// loop has a simple "messages → reply (string), cost-or-cached" call.
// We collect chunks from Stream and concatenate; for backtest there is
// no UI to update incrementally so streaming is just a wire-level
// detail.
type llmRunner struct {
	client      *deepseek.Client
	cache       *fileCache
	model       string
	temperature *float64
	seed        *int
}

func newLLMRunner(client *deepseek.Client, cache *fileCache, modelName string, temp float64, seed int) *llmRunner {
	t := temp
	s := seed
	return &llmRunner{
		client:      client,
		cache:       cache,
		model:       modelName,
		temperature: &t,
		seed:        &s,
	}
}

// Run sends the message array; on cache hit, returns immediately. On
// miss, calls DeepSeek, caches the full reply text, and returns it.
// The bool reports whether the call hit the cache (the caller uses it
// to track cost).
func (r *llmRunner) Run(ctx context.Context, turns []model.Turn) (reply string, cached bool, err error) {
	key := r.cache.keyFor(r.model, r.temperature, r.seed, turns)
	if cached, ok := r.cache.Get(key); ok {
		return cached.Reply, true, nil
	}

	chunks, errCh := r.client.Stream(ctx, turns)
	var b strings.Builder
	for c := range chunks {
		b.WriteString(c)
	}
	if streamErr, ok := <-errCh; ok && streamErr != nil {
		return "", false, streamErr
	}
	full := b.String()
	tempVal := 0.0
	if r.temperature != nil {
		tempVal = *r.temperature
	}
	seedVal := 0
	if r.seed != nil {
		seedVal = *r.seed
	}
	if err := r.cache.Put(key, cachedResponse{
		Reply:       full,
		Model:       r.model,
		Temperature: tempVal,
		Seed:        seedVal,
	}); err != nil {
		// Cache failures aren't fatal — we already have the reply.
		// Skipping persistence just means re-paying next run.
	}
	return full, false, nil
}

// Compile-time check: the runner is invoked with biz.LLMProvider-shaped
// turns even though we use the Stream method directly. Keeping the
// import live makes the dependency obvious to readers.
var _ biz.LLMProvider = (*deepseek.Client)(nil)
