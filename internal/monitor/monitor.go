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
	State map[string]Status
}

func NewMonitor() *Monitor {
	return &Monitor{State: make(map[string]Status)}
}

func (m *Monitor) Update(r checker.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// read-modify-write-back pattern
	s := m.State[r.URL]
	s.LastChecked = time.Now()
	s.LastStatusCode = r.StatusCode
	if r.Err != nil {
		s.ConsecutiveFailures++
		s.LastError = r.Err
	} else {
		s.ConsecutiveFailures = 0
		s.LastSuccess = time.Now()
		s.LastError = nil
	}
	m.State[r.URL] = s
}

func (m *Monitor) Snapshot() map[string]Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]Status, len(m.State))
	for k, v := range m.State {
		out[k] = v
	}
	return out
}
