package checker

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type Result struct {
	URL        string
	StatusCode int
	Duration   time.Duration
	Err        error
}

func CheckWebsite(ctx context.Context, url string) Result {
	var result Result
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Result{URL: url, StatusCode: 0, Duration: time.Since(start), Err: err}
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{URL: url, StatusCode: 0, Duration: time.Since(start), Err: err}
	}
	// the defer should be after the error checking because
	//if the response was nil the code wont panic.
	defer response.Body.Close()
	result = Result{
		URL:        url,
		StatusCode: response.StatusCode,
		Duration:   time.Since(start),
		Err:        nil,
	}
	return result
}

const maxWorkers = 5

// We wanna return the channel so it should be recieve-only <-
func CheckAll(ctx context.Context, urls []string) <-chan Result {
	results := make(chan Result)
	jobs := make(chan string)
	var wg sync.WaitGroup
	// 5 workers, 5 urls or 500 urls, only 5 CheckWebsite(u)
	//could call at once <- bounded concurrency
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				r := CheckWebsite(ctx, u)
				select {
				case results <- r:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	// Feeder, jobs is unbuffered so it blocks untill a worker
	//is ready to take it. thats our backpressure.
	//the feeder's pace depends on the worker's pace.
	go func() {
		defer close(jobs)
		for _, u := range urls {
			select {
			case jobs <- u:
			case <-ctx.Done():
				return
			}
		}
	}()
	// Closer
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}
