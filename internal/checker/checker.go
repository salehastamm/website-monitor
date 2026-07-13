package checker

import (
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

func CheckWebsite(url string) Result {
	var result Result
	start := time.Now()
	response, err := http.Get(url)
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
func CheckAll(urls []string) <-chan Result {

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
				results <- CheckWebsite(u)
			}
		}()
	}
	// Feeder, jobs is unbuffered so it blocks untill a worker
	//is ready to take it. thats our backpressure.
	//the feeder's pace depends on the worker's pace.
	go func() {
		for _, u := range urls {
			jobs <- u
		}
		close(jobs) // whoever send is who closes
	}()
	// Closer
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}
