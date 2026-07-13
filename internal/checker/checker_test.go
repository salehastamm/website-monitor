package checker

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckWebsite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	result := CheckWebsite(srv.URL)

	if result.Err != nil {
		t.Errorf("expected no error, got %v", result.Err)
	}
	if result.StatusCode != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, result.StatusCode)
	}
}

func TestCheckWebsiteFailure(t *testing.T) {
	// Start a server just to get a valid address, then kill it immediately
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := srv.URL
	srv.Close()
	// nothing is listening at deadURL anymore, so the GET must fail
	// at the transport layar(connection refused)
	result := CheckWebsite(deadURL)
	if result.Err == nil {
		t.Errorf("expected an error for a dead server, got nil")
	}
	if result.StatusCode != 0 {
		t.Errorf("expected status code 0 on failure, got %d", result.StatusCode)
	}
}

func TestCheckAll(t *testing.T) {
	srv200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv200.Close()

	srv404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv404.Close()

	srv503 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv503.Close()

	urls := []string{srv200.URL, srv404.URL, srv503.URL}

	// Receive results from the channel.
	got := make(map[string]int)
	count := 0

	for r := range CheckAll(urls) {
		count++
		got[r.URL] = r.StatusCode
	}

	if count != len(urls) {
		t.Fatalf("expected %d results, got %d", len(urls), count)
	}

	if got[srv200.URL] != http.StatusOK {
		t.Errorf("expected %s to return %d, got %d",
			srv200.URL, http.StatusOK, got[srv200.URL])
	}

	if got[srv404.URL] != http.StatusNotFound {
		t.Errorf("expected %s to return %d, got %d",
			srv404.URL, http.StatusNotFound, got[srv404.URL])
	}

	if got[srv503.URL] != http.StatusServiceUnavailable {
		t.Errorf("expected %s to return %d, got %d",
			srv503.URL, http.StatusServiceUnavailable, got[srv503.URL])
	}
}

func TestCheckAllBoundedConcurrency(t *testing.T) {
	//How many requests are inside the handler right now.
	var active int32
	//The highest value active ever reached.
	var peak int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// n is just the value of active at the instant
		n := atomic.AddInt32(&active, 1) // two goroutines touching shared variable is dangerous.
		// we could used mutex instead we used atomic to ensure safe operation

		for {
			old := atomic.LoadInt32(&peak) //current peak
			if n <= old {
				break // peak already >= n
			}
			if atomic.CompareAndSwapInt32(&peak, old, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	const numURLs = 20
	urls := make([]string, numURLs)
	for i := range urls {
		urls[i] = srv.URL
	}

	//Drain every result, dont care about the values
	for range CheckAll(urls) {
	}

	//Concurrency never exceeded pool size
	if got := atomic.LoadInt32(&peak); got > maxWorkers {
		t.Errorf("expected at most %d concurrent requests, but peak was %d", maxWorkers, got)
	}
}
