package store

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/salehastamm/website-monitor/internal/ledger"
	"github.com/salehastamm/website-monitor/internal/monitor"
)

// ---- helpers ----------------------------------------------------------------

func tmpFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "store_test_*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	os.Remove(f.Name()) // we only want the path, not the file itself
	return f.Name()
}

// ---- SaveState / LoadState --------------------------------------------------

func TestSaveLoadState_roundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond) // JSON only stores ms precision
	state := map[string]monitor.Status{
		"https://example.com": {
			LastStatusCode:      200,
			LastChecked:         now,
			ConsecutiveFailures: 0,
			LastSuccess:         now,
			LastHash:            "abc123",
			LastChanged:         now,
			Changed:             false,
		},
		"https://down.example": {
			LastStatusCode:      0,
			LastChecked:         now,
			ConsecutiveFailures: 3,
			LastError:           errors.New("connection refused"),
		},
	}

	path := tmpFile(t)
	if err := SaveState(path, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	got, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if len(got) != len(state) {
		t.Fatalf("got %d entries, want %d", len(got), len(state))
	}
	for url, want := range state {
		g, ok := got[url]
		if !ok {
			t.Errorf("missing entry for %s", url)
			continue
		}
		if g.LastStatusCode != want.LastStatusCode {
			t.Errorf("%s: LastStatusCode = %d, want %d", url, g.LastStatusCode, want.LastStatusCode)
		}
		if g.ConsecutiveFailures != want.ConsecutiveFailures {
			t.Errorf("%s: ConsecutiveFailures = %d, want %d", url, g.ConsecutiveFailures, want.ConsecutiveFailures)
		}
		if g.LastHash != want.LastHash {
			t.Errorf("%s: LastHash = %q, want %q", url, g.LastHash, want.LastHash)
		}
		// Error is serialized as a string; check message equality.
		wantErrMsg := ""
		if want.LastError != nil {
			wantErrMsg = want.LastError.Error()
		}
		gotErrMsg := ""
		if g.LastError != nil {
			gotErrMsg = g.LastError.Error()
		}
		if gotErrMsg != wantErrMsg {
			t.Errorf("%s: LastError = %q, want %q", url, gotErrMsg, wantErrMsg)
		}
	}
}

func TestLoadState_missingFile(t *testing.T) {
	got, err := LoadState(tmpFile(t)) // file was deleted by tmpFile
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestSaveState_isAtomic(t *testing.T) {
	// A second Save should fully replace the first.
	path := tmpFile(t)

	state1 := map[string]monitor.Status{"https://a.com": {LastStatusCode: 200}}
	state2 := map[string]monitor.Status{"https://b.com": {LastStatusCode: 503}}

	if err := SaveState(path, state1); err != nil {
		t.Fatal(err)
	}
	if err := SaveState(path, state2); err != nil {
		t.Fatal(err)
	}

	got, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["https://a.com"]; ok {
		t.Error("stale entry from first save should not be present")
	}
	if _, ok := got["https://b.com"]; !ok {
		t.Error("entry from second save should be present")
	}
}

// ---- AppendBlock / LoadChain ------------------------------------------------

func makeChain(t *testing.T, n int) *ledger.Chain {
	t.Helper()
	c := ledger.NewChain()
	for i := 0; i < n; i++ {
		root := make([]byte, 32)
		root[0] = byte(i + 1)
		c.Append(root, time.Now())
	}
	return c
}

func TestAppendLoadChain_roundtrip(t *testing.T) {
	c := makeChain(t, 4)
	snap := c.Snapshot()

	path := tmpFile(t)
	for _, b := range snap {
		if err := AppendBlock(path, b); err != nil {
			t.Fatalf("AppendBlock: %v", err)
		}
	}

	got, err := LoadChain(path)
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(got) != len(snap) {
		t.Fatalf("got %d blocks, want %d", len(got), len(snap))
	}
	for i, want := range snap {
		g := got[i]
		if g.Index != want.Index {
			t.Errorf("block %d: Index = %d, want %d", i, g.Index, want.Index)
		}
		if string(g.Hash) != string(want.Hash) {
			t.Errorf("block %d: Hash mismatch", i)
		}
		if string(g.PrevHash) != string(want.PrevHash) {
			t.Errorf("block %d: PrevHash mismatch", i)
		}
		if string(g.MerkleRoot) != string(want.MerkleRoot) {
			t.Errorf("block %d: MerkleRoot mismatch", i)
		}
	}
}

func TestLoadChain_missingFile(t *testing.T) {
	got, err := LoadChain(tmpFile(t))
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %v", got)
	}
}

func TestLoadChain_restoredChainValidates(t *testing.T) {
	c := makeChain(t, 5)
	snap := c.Snapshot()

	path := tmpFile(t)
	for _, b := range snap {
		if err := AppendBlock(path, b); err != nil {
			t.Fatal(err)
		}
	}

	blocks, err := LoadChain(path)
	if err != nil {
		t.Fatal(err)
	}

	restored := ledger.NewChainFromBlocks(blocks)
	if !restored.Validate() {
		t.Error("restored chain failed Validate()")
	}
}
