package telegram

import (
	"context"
	"time"
)

// KeepTyping fires sendChatAction("typing") periodically until the returned
// cancel function is invoked. Telegram's typing indicator expires after ~5s
// so we refresh it every 4s.
//
// Usage:
//
//	stop := telegram.KeepTyping(ctx, bot, chatID)
//	defer stop()
func KeepTyping(ctx context.Context, bot *AdvisorBot, chatID int64) (stop func()) {
	innerCtx, cancel := context.WithCancel(ctx)

	// Fire immediately so the user sees "typing..." without waiting 4s.
	_ = bot.SendChatAction(innerCtx, chatID, "typing")

	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-innerCtx.Done():
				return
			case <-ticker.C:
				_ = bot.SendChatAction(innerCtx, chatID, "typing")
			}
		}
	}()

	return cancel
}
