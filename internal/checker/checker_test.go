package checker

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
	results := CheckAll(urls)
	if len(results) != len(urls) {
		t.Errorf("Expected the same size for results and urls, got different size")
	}
	expected := []int{200, 404, 503}
	for i, result := range results {
		if result.StatusCode != expected[i] {
			t.Errorf("Expected Status %d got Status %d", expected[i], result.StatusCode)
		}
	}
}
