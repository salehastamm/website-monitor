package checker

import (
	"context"
	"errors"
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

	result := CheckWebsite(context.Background(), srv.URL)

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
	result := CheckWebsite(context.Background(), deadURL)
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

	for r := range CheckAll(context.Background(), urls) {
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
	for range CheckAll(context.Background(), urls) {
	}

	//Concurrency never exceeded pool size
	if got := atomic.LoadInt32(&peak); got > maxWorkers {
		t.Errorf("expected at most %d concurrent requests, but peak was %d", maxWorkers, got)
	}
	// floor: the pool actually ran in parallel, not sequentially
	if got := atomic.LoadInt32(&peak); got < 2 {
		t.Errorf("expected concurrent requests, but peak was only %d", got)
	}
}

// When the server is slow, does it actually give up?
func TestCheckWebsiteTimeout(t *testing.T) {
	// a fake website that pauses 500ms before answering
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// give up after 50ms
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	result := CheckWebsite(ctx, srv.URL) // Check the slow site with 50ms give up rule
	elapsed := time.Since(start)

	// check A: it should have FAILED
	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Errorf("expected a timeout error, got %v", result.Err)
	}

	// check B : it shoud have given up after 50ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("expected it to give up quickly, but it took %v", elapsed)
	}
}

// When we say STOP!, does it actually stop?
func TestCheckAllCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 20 copies of the slow URL, enough to keep all workers busy.
	urls := make([]string, 20)
	for i := range urls {
		urls[i] = srv.URL
	}

	//a context we can cancel ourselves
	ctx, cancel := context.WithCancel(context.Background())

	results := CheckAll(ctx, urls)
	cancel()

	start := time.Now()
	for range results { // drain untill the channel closes
	}
	elapsed := time.Since(start)
	if elapsed > 1*time.Second {
		t.Errorf("expected fast shutdown after cancel, took %v", elapsed)
	}
}

func TestCheckWithRetrySucceedsAfterFailures(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable) //503 on tries 1 and 2 so we can retry on it
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result := CheckWithRetry(context.Background(), srv.URL, 5) //allows up to 5 tries

	if result.Err != nil {
		t.Errorf("expected success, got error: %v", result.Err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", got)
	}
}

func TestCheckWithRetryGivesUp(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	result := CheckWithRetry(context.Background(), srv.URL, 3)

	if result.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected the final failing result (503), got %d", result.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", got)
	}
}

// this ensures that if a cancel arrives mid-backoff, it returns fast,
// not after sleeping out the full backoff
func TestCheckWithRetryCancelBeatsBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// cancel shortly after starting while its mid-backoff
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	CheckWithRetry(ctx, srv.URL, 10)
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Errorf("expected cancel to cut the backoff short, but it took %v", elapsed)
	}
}
