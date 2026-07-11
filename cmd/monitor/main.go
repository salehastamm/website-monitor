package main

import (
	"fmt"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
)

func main() {
	urls := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
		"https://invalid.example",
	}
	start := time.Now()

	for result := range checker.CheckAll(urls) {
		if result.Err != nil {
			fmt.Printf("%s failed: %v\n", result.URL, result.Err)
		} else {
			fmt.Printf("%s returned %d in %v\n",
				result.URL,
				result.StatusCode,
				result.Duration)
		}
	}

	fmt.Println("Finished in", time.Since(start))
}
