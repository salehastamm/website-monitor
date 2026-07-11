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

// We wanna return the channel so it should be recieve-only <-
func CheckAll(urls []string) <-chan Result {
	ch := make(chan Result)

	var wg sync.WaitGroup

	for _, u := range urls {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()
			ch <- CheckWebsite(url)
		}(u)
	}
	// Because we return channel and the receiver is main
	// we put wg.Wait() and close(ch) inside a goroutine so
	// the CheckAll function could return ch any time its ready
	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}
