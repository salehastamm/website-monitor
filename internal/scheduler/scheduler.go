package scheduler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
	"github.com/salehastamm/website-monitor/internal/ledger"
	"github.com/salehastamm/website-monitor/internal/merkle"
	"github.com/salehastamm/website-monitor/internal/monitor"
	"github.com/salehastamm/website-monitor/internal/store"
)

// Paths holds the file paths used for persistence. Set either field to ""
// to skip persisting that piece of state.
type Paths struct {
	State string // path to monitor_state.json
	Chain string // path to ledger.jsonl
}

func Run(ctx context.Context, mon *monitor.Monitor, chain *ledger.Chain, urls []string, interval time.Duration, paths Paths) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	runOnce(ctx, mon, chain, urls, paths) // Immediate first check
	for {
		select {
		case <-ticker.C:
			runOnce(ctx, mon, chain, urls, paths)
		case <-ctx.Done():
			return
		}
	}
}

func runOnce(ctx context.Context, mon *monitor.Monitor, chain *ledger.Chain, urls []string, paths Paths) {

	roundHashes := make(map[string]string)

	for result := range checker.CheckAll(ctx, urls) {
		mon.Update(result)
		if result.BodyHash != "" {
			roundHashes[result.URL] = result.BodyHash
		}
	}

	if len(roundHashes) == 0 {
		return
	}

	leaves := merkle.BuildLeaves(roundHashes)
	root := merkle.Root(leaves)

	fmt.Printf("[Scheduler] Round observed %d sites. Merkle root: %x\n", len(roundHashes), root)

	// chain is optional; callers that don't need a ledger pass nil
	if chain != nil {
		block := chain.Append(root, time.Now())
		fmt.Printf("[Ledger] Chain Length: %d | Tip Hash: %x\n", block.Index+1, block.Hash)

		// Persist the new block by appending it to the chain file.
		if paths.Chain != "" {
			if err := store.AppendBlock(paths.Chain, block); err != nil {
				fmt.Fprintf(os.Stderr, "[Store] Failed to persist block: %v\n", err)
			}
		}
	}

	// Persist the monitor state snapshot after every round.
	if paths.State != "" {
		if err := store.SaveState(paths.State, mon.Snapshot()); err != nil {
			fmt.Fprintf(os.Stderr, "[Store] Failed to persist state: %v\n", err)
		}
	}
}
