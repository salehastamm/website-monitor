package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
)

func main() {
	var wg sync.WaitGroup
	URL := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
		"https://invalid.example",
	}
	results := make([]checker.Result, len(URL))
	start := time.Now()
	for i, u := range URL {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			results[i] = checker.CheckWebsite(u)
		}(i, u)
	}
	wg.Wait()
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
