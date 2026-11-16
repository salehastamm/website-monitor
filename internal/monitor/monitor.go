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
	LastHash            string
	LastChanged         time.Time
	Changed             bool
}

type Monitor struct {
	mu    sync.Mutex
	state map[string]Status
}

func NewMonitor() *Monitor {
	return &Monitor{state: make(map[string]Status)}
}

// Seed replaces the internal state with the provided snapshot. It is intended
// for restoring persisted state at startup and must be called before any
// concurrent updates begin.
func (m *Monitor) Seed(state map[string]Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = make(map[string]Status, len(state))
	for k, v := range state {
		m.state[k] = v
	}
}

func (m *Monitor) Update(r checker.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// read-modify-write-back pattern
	now := time.Now()
	s := m.state[r.URL]
	s.Changed = false
	if r.BodyHash != "" {
		if s.LastHash != "" && r.BodyHash != s.LastHash {
			s.Changed = true
			s.LastChanged = now
		}
		s.LastHash = r.BodyHash
	}
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
