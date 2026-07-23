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
	// create a child context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// build a GET request and attach context to it
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Result{URL: url, StatusCode: 0, Duration: time.Since(start), Err: err}
	}
	// send the built request
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
const maxAttempts = 3

// We wanna return the channel so it should be recieve-only <-
func CheckAll(ctx context.Context, urls []string) <-chan Result {
	results := make(chan Result)
	jobs := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				r := CheckWithRetry(ctx, u, maxAttempts)
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
			//Also a cancellation to avoid goroutines leaking
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

func isRetryable(r Result) bool {
	if r.Err != nil || r.StatusCode >= 500 {
		return true
	}
	return false
}

func CheckWithRetry(ctx context.Context, url string, maxAttempts int) Result {
	var result Result
	backoff := 100 * time.Millisecond
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result = CheckWebsite(ctx, url)
		if !isRetryable(result) {
			return result
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return result
		}
		backoff *= 2
	}
	return result
}
