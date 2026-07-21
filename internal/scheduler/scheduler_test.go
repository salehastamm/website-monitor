package scheduler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/salehastamm/website-monitor/internal/monitor"
)

func TestRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	mon := monitor.NewMonitor()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	Run(ctx, mon, []string{srv.URL}, 20*time.Millisecond)

	if got := atomic.LoadInt32(&hits); got < 2 {
		t.Errorf("expected multiple rounds (>1 hit), got %d", got)
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mon := monitor.NewMonitor()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		Run(ctx, mon, []string{srv.URL}, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop after cancel - its leaking/hung")
	}
}
