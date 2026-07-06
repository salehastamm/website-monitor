package main

import (
	"fmt"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
)

func main() {
	URL := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
		"https://invalid.example",
	}
	start := time.Now()
	for _, u := range URL {
		result := checker.CheckWebsite(u)
		if result.Err != nil {
			fmt.Printf("%s failed : %s in %v \n", result.URL, result.Err, result.Duration)
		} else {
			fmt.Printf("%s returned %d in %v\n", result.URL, result.StatusCode, result.Duration)
		}
	}
	finished := time.Since(start)
	fmt.Printf("Monitoring Websites Sequentially %v\n", finished)
}
