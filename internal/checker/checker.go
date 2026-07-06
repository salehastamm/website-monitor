package checker

import (
	"net/http"
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
