// internal/poller/runner.go
package poller

import (
	"context"
	"time"
)

// Run starts the ticker loop and emits PollResult on the provided channel.
// One goroutine per unit. No overlap. No retries.
func (p *Poller) Run(ctx context.Context, out chan<- PollResult) {
	ticker := time.NewTicker(p.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			out <- p.PollOnce()
		}
	}
}
