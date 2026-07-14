package monitor

import (
	"errors"
	"sync"
	"testing"

	"github.com/salehastamm/website-monitor/internal/checker"
)

func TestUpdate(t *testing.T) {
	var wg sync.WaitGroup
	m := NewMonitor()
	r := checker.Result{
		URL:        "https://example.com",
		StatusCode: 200,
		Err:        nil,
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.Update(r)
			}
		}()
	}
	wg.Wait()
	snap := m.Snapshot()
	url := "https://example.com"
	if snap[url].LastStatusCode != 200 || snap[url].ConsecutiveFailures != 0 {
		t.Errorf("unexpected final state: %+v", snap[url])
	}
}

func TestUpdateConsecutiveFailures(t *testing.T) {
	m := NewMonitor()
	url := "https://example.com"
	m.Update(checker.Result{URL: url,
		Err: errors.New("Boom")})
	m.Update(checker.Result{URL: url,
		Err: errors.New("Boom")})
	m.Update(checker.Result{URL: url,
		Err: errors.New("Boom")})
	snap := m.Snapshot()
	if snap[url].ConsecutiveFailures != 3 {
		t.Errorf("expected %d got %d", 3, snap[url].ConsecutiveFailures)
	}
	m.Update(checker.Result{URL: url,
		StatusCode: 200,
		Err:        nil})
	snap = m.Snapshot()
	if snap[url].ConsecutiveFailures != 0 {
		t.Errorf("Expected 0 got %d", snap[url].ConsecutiveFailures)
	}
	if snap[url].LastSuccess.IsZero() {
		t.Errorf("Expected Successful but got %v", snap[url].LastSuccess)
	}
}
