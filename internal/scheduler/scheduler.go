package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/salehastamm/website-monitor/internal/checker"
	"github.com/salehastamm/website-monitor/internal/ledger"
	"github.com/salehastamm/website-monitor/internal/merkle"
	"github.com/salehastamm/website-monitor/internal/monitor"
)

func Run(ctx context.Context, mon *monitor.Monitor, chain *ledger.Chain, urls []string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	runOnce(ctx, mon, chain, urls) //Immediate first Check
	for {
		select {
		case <-ticker.C:
			runOnce(ctx, mon, chain, urls)
		case <-ctx.Done():
			return
		}
	}
}

func runOnce(ctx context.Context, mon *monitor.Monitor, chain *ledger.Chain, urls []string) {

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
	}
}
