package scheduler

import (
	"context"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
	"github.com/salehastamm/website-monitor/internal/monitor"
)

func Run(ctx context.Context, mon *monitor.Monitor, urls []string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// Immediate first check
	for result := range checker.CheckAll(ctx, urls) {
		mon.Update(result)
	}
	for {
		select {
		case <-ticker.C:
			for result := range checker.CheckAll(ctx, urls) {
				mon.Update(result)
			}
		case <-ctx.Done():
			return
		}
	}
}
