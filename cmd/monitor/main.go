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
	results := checker.CheckAll(urls)
	for i := 0; i < len(results); i++ {
		if results[i].Err != nil {
			fmt.Printf("%s failed : %s in %v \n", results[i].URL, results[i].Err, results[i].Duration)
		} else {
			fmt.Printf("%s returned %d in %v\n", results[i].URL, results[i].StatusCode, results[i].Duration)
		}
	}
	finished := time.Since(start)
	fmt.Printf("Monitoring Websites Concurrently %v\n", finished)
}
