package main

import (
	"context"
	"fmt"
	"time"

	"github.com/salehastamm/website-monitor/internal/monitor"
	"github.com/salehastamm/website-monitor/internal/scheduler"
)

func main() {
	mon := monitor.NewMonitor()
	urls := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
		"https://invalid.example",
	}
	start := time.Now()
	serviceTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), serviceTimeout)
	defer cancel()
	interval := 10 * time.Second
	scheduler.Run(ctx, mon, urls, interval)
	for url, status := range mon.Snapshot() {
		if status.LastError != nil {
			fmt.Printf("%s DOWN (%d consecutive failures)\n", url, status.ConsecutiveFailures)
		} else {
			fmt.Printf("%s UP %d\n", url, status.LastStatusCode)
		}
	}

	fmt.Println("Finished in", time.Since(start))
}
