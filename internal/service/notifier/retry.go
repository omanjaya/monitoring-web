package notifier

import (
	"time"

	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

// retryWithBackoff attempts fn up to maxRetries times with exponential backoff.
// Initial delay is 2 seconds, doubling each retry (2s, 4s, 8s).
func retryWithBackoff(name string, maxRetries int, fn func() error) error {
	var lastErr error
	delay := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			if attempt > 0 {
				logger.Info().Str("notifier", name).Int("attempt", attempt+1).Msg("Notification sent after retry")
			}
			return nil
		}

		if attempt < maxRetries {
			logger.Warn().Str("notifier", name).Int("attempt", attempt+1).Err(lastErr).Dur("retry_in", delay).Msg("Notification failed, retrying")
			time.Sleep(delay)
			delay *= 2
		}
	}

	logger.Error().Str("notifier", name).Int("max_retries", maxRetries).Err(lastErr).Msg("Notification failed after all retries")
	return lastErr
}
