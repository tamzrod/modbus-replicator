// internal/poller/runner.go
package poller

import (
	"context"
	"log"
	"time"
)

func (p *Poller) Run(ctx context.Context, out chan<- PollResult) {
	log.Println("poller: started")

	ticker := time.NewTicker(p.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("poller: context done")
			return

		case <-ticker.C:
			res := p.PollOnce()

			// NOTE:
			// Per-tick success logging intentionally removed.
			// Errors are surfaced via status memory and downstream handling.
			// Silence on success prevents log flooding at scale.

			out <- res
		}
	}
}
