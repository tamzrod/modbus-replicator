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

			log.Printf(
				"poller: tick unit=%s blocks=%d err=%v",
				p.cfg.UnitID,
				len(res.Blocks),
				res.Err,
			)

			out <- res
		}
	}
}
