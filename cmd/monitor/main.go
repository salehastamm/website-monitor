package main

import (
	"fmt"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
	"github.com/salehastamm/website-monitor/internal/monitor"
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

	for result := range checker.CheckAll(urls) {
		mon.Update(result)
	}
	for url, status := range mon.Snapshot() {
		if status.LastError != nil {
			fmt.Printf("%s DOWN (%d consecutive failures)\n", url, status.ConsecutiveFailures)
		} else {
			fmt.Printf("%s UP %d\n", url, status.LastStatusCode)
		}
	}

	fmt.Println("Finished in", time.Since(start))
}
