package telegram

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// StartAdvisorListener runs a long-polling loop against the AdvisorBot and
// pushes every received Update into the returned channel. The caller owns
// the lifecycle: cancel `ctx` to stop the loop; the channel is closed when
// the goroutine exits.
//
// Failure strategy: transient network errors back off linearly (1s, 2s, ...
// up to 30s) and retry forever; callers should not restart the listener.
func StartAdvisorListener(ctx context.Context, bot *AdvisorBot) <-chan Update {
	out := make(chan Update, 128)

	go func() {
		defer close(out)

		var offset int64
		const pollTimeoutSec = 30

		// Backoff on transient errors to avoid hammering Telegram.
		backoff := time.Second
		const maxBackoff = 30 * time.Second

		log.Info().Msg("advisor listener: long-polling loop started")

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("advisor listener: shutting down")
				return
			default:
			}

			updates, err := bot.GetUpdates(ctx, offset, pollTimeoutSec)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Warn().Err(err).Dur("backoff", backoff).Msg("advisor listener: getUpdates failed, retrying")
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return
				}
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			backoff = time.Second

			for _, u := range updates {
				if u.UpdateID >= offset {
					offset = u.UpdateID + 1
				}
				if u.Message == nil {
					continue
				}
				select {
				case out <- u:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out
}
