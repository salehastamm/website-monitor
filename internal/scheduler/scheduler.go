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
	runOnce(ctx, mon, urls) //Immediate first Check
	for {
		select {
		case <-ticker.C:
			runOnce(ctx, mon, urls)
		case <-ctx.Done():
			return
		}
	}
}

func runOnce(ctx context.Context, mon *monitor.Monitor, urls []string) {
	for result := range checker.CheckAll(ctx, urls) {
		mon.Update(result)
	}
}
