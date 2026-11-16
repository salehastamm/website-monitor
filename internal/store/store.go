package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/salehastamm/website-monitor/internal/ledger"
	"github.com/salehastamm/website-monitor/internal/monitor"
)

// statusJSON is the JSON-serializable form of monitor.Status.
// The error field is flattened to a string since the error interface
// cannot be directly marshaled by encoding/json.
type statusJSON struct {
	LastStatusCode      int       `json:"last_status_code"`
	LastError           string    `json:"last_error,omitempty"`
	LastChecked         time.Time `json:"last_checked"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastSuccess         time.Time `json:"last_success"`
	LastHash            string    `json:"last_hash"`
	LastChanged         time.Time `json:"last_changed"`
	Changed             bool      `json:"changed"`
}

func toJSON(s monitor.Status) statusJSON {
	sj := statusJSON{
		LastStatusCode:      s.LastStatusCode,
		LastChecked:         s.LastChecked,
		ConsecutiveFailures: s.ConsecutiveFailures,
		LastSuccess:         s.LastSuccess,
		LastHash:            s.LastHash,
		LastChanged:         s.LastChanged,
		Changed:             s.Changed,
	}
	if s.LastError != nil {
		sj.LastError = s.LastError.Error()
	}
	return sj
}

func fromJSON(sj statusJSON) monitor.Status {
	s := monitor.Status{
		LastStatusCode:      sj.LastStatusCode,
		LastChecked:         sj.LastChecked,
		ConsecutiveFailures: sj.ConsecutiveFailures,
		LastSuccess:         sj.LastSuccess,
		LastHash:            sj.LastHash,
		LastChanged:         sj.LastChanged,
		Changed:             sj.Changed,
	}
	if sj.LastError != "" {
		s.LastError = errors.New(sj.LastError)
	}
	return s
}

// SaveState atomically writes the monitor state snapshot to a JSON file,
// replacing any previous contents. It uses a write-to-temp-then-rename
// strategy to avoid leaving a partial file on disk if the process crashes.
func SaveState(path string, state map[string]monitor.Status) error {
	jMap := make(map[string]statusJSON, len(state))
	for url, s := range state {
		jMap[url] = toJSON(s)
	}

	// Write to a temp file in the same directory so the rename is atomic.
	dir := dirOf(path)
	f, err := os.CreateTemp(dir, ".monitor_state_*.json.tmp")
	if err != nil {
		return fmt.Errorf("store.SaveState: create temp: %w", err)
	}
	tmpName := f.Name()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(jMap); encErr != nil {
		f.Close()
		os.Remove(tmpName)
		return fmt.Errorf("store.SaveState: encode: %w", encErr)
	}
	if closeErr := f.Close(); closeErr != nil {
		os.Remove(tmpName)
		return fmt.Errorf("store.SaveState: close temp: %w", closeErr)
	}
	if renameErr := os.Rename(tmpName, path); renameErr != nil {
		os.Remove(tmpName)
		return fmt.Errorf("store.SaveState: rename: %w", renameErr)
	}
	return nil
}

// LoadState reads a monitor state snapshot from a JSON file and returns
// the URL → Status map. If the file does not exist yet, an empty map is
// returned — that is not treated as an error.
func LoadState(path string) (map[string]monitor.Status, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]monitor.Status), nil
	}
	if err != nil {
		return nil, fmt.Errorf("store.LoadState: open: %w", err)
	}
	defer f.Close()

	var jMap map[string]statusJSON
	if decErr := json.NewDecoder(f).Decode(&jMap); decErr != nil {
		return nil, fmt.Errorf("store.LoadState: decode: %w", decErr)
	}

	out := make(map[string]monitor.Status, len(jMap))
	for url, sj := range jMap {
		out[url] = fromJSON(sj)
	}
	return out, nil
}

// AppendBlock serializes block as a single JSON line and appends it to path,
// creating the file if it does not exist. Using one JSON object per line
// (JSON Lines / NDJSON) makes the file easy to tail and append without
// rewriting the whole file.
func AppendBlock(path string, block ledger.Block) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("store.AppendBlock: open: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("store.AppendBlock: marshal: %w", err)
	}
	line = append(line, '\n')
	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("store.AppendBlock: write: %w", err)
	}
	return nil
}

// LoadChain reads all blocks from a JSON Lines file and returns them in
// order. If the file does not exist yet, a nil slice is returned — that
// is not treated as an error.
func LoadChain(path string) ([]ledger.Block, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store.LoadChain: open: %w", err)
	}
	defer f.Close()

	var blocks []ledger.Block
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // skip blank lines
		}
		var b ledger.Block
		if err := json.Unmarshal(line, &b); err != nil {
			return nil, fmt.Errorf("store.LoadChain: line %d: %w", lineNum, err)
		}
		blocks = append(blocks, b)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("store.LoadChain: scan: %w", err)
	}
	return blocks, nil
}

// dirOf returns the directory component of a file path, or "." if path has
// no directory separator.
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
