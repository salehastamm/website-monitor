package monitor

import (
	"sync"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
)

type Status struct {
	LastStatusCode      int
	LastError           error
	LastChecked         time.Time
	ConsecutiveFailures int
	LastSuccess         time.Time
}

type Monitor struct {
	mu    sync.Mutex
	state map[string]Status
}

func NewMonitor() *Monitor {
	return &Monitor{state: make(map[string]Status)}
}

func (m *Monitor) Update(r checker.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// read-modify-write-back pattern
	now := time.Now()
	s := m.state[r.URL]
	s.LastChecked = now
	s.LastStatusCode = r.StatusCode
	if r.Err != nil {
		s.ConsecutiveFailures++
		s.LastError = r.Err
	} else {
		s.ConsecutiveFailures = 0
		s.LastSuccess = now
		s.LastError = nil
	}
	m.state[r.URL] = s
}

func (m *Monitor) Snapshot() map[string]Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	// make a copy of the main map and return it
	out := make(map[string]Status, len(m.state))
	for k, v := range m.state {
		out[k] = v
	}
	return out
}
