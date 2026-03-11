// internal/poller/runner.go
package poller

import (
	"context"
	"log"
	"time"
)

// Run starts the per-read scheduler loop.
// Each read block fires independently at its own interval.
// Reads are executed sequentially; no two reads run concurrently.
func (p *Poller) Run(ctx context.Context, out chan<- PollResult) {
	log.Println("poller: started")

	for {
		select {
		case <-ctx.Done():
			log.Println("poller: context done")
			return
		default:
		}

		now := time.Now()

		// nextWake tracks the earliest time any read is due after this iteration.
		nextWake := now.Add(time.Hour)

		for i := range p.schedules {
			if now.Before(p.schedules[i].nextExec) {
				if p.schedules[i].nextExec.Before(nextWake) {
					nextWake = p.schedules[i].nextExec
				}
				continue
			}

			res := p.executeSingleRead(p.schedules[i].cfg)
			out <- res

			p.schedules[i].nextExec = now.Add(p.schedules[i].cfg.Interval)

			// Update nextWake now that this schedule has been advanced.
			if p.schedules[i].nextExec.Before(nextWake) {
				nextWake = p.schedules[i].nextExec
			}
		}

		// Sleep until the next read is due, bounded by a 1ms floor to prevent
		// busy-spinning if nextWake is in the past due to clock jitter.
		sleep := time.Until(nextWake)
		if sleep < time.Millisecond {
			sleep = time.Millisecond
		}

		select {
		case <-ctx.Done():
			log.Println("poller: context done")
			return
		case <-time.After(sleep):
		}
	}
}
