package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/salehastamm/website-monitor/internal/ledger"
	"github.com/salehastamm/website-monitor/internal/monitor"
	"github.com/salehastamm/website-monitor/internal/scheduler"
	"github.com/salehastamm/website-monitor/internal/store"
)

const (
	statePath = "monitor_state.json"
	chainPath = "ledger.jsonl"
)

func main() {
	// ── Restore ledger chain ──────────────────────────────────────────────────
	blocks, err := store.LoadChain(chainPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load ledger: %v\n", err)
		os.Exit(1)
	}

	var chain *ledger.Chain
	if len(blocks) > 0 {
		chain = ledger.NewChainFromBlocks(blocks)
		if !chain.Validate() {
			fmt.Fprintln(os.Stderr, "❌ CRITICAL: Loaded ledger failed validation — possible tampering!")
			os.Exit(1)
		}
		fmt.Printf("[Startup] Loaded %d blocks from %s — chain is valid ✅\n", len(blocks), chainPath)
	} else {
		chain = ledger.NewChain()
		fmt.Println("[Startup] No existing ledger found, starting a fresh chain.")
	}

	// ── Restore monitor state ─────────────────────────────────────────────────
	state, err := store.LoadState(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load monitor state: %v\n", err)
		os.Exit(1)
	}

	mon := monitor.NewMonitor()
	if len(state) > 0 {
		mon.Seed(state)
		fmt.Printf("[Startup] Restored prior status for %d site(s) from %s\n", len(state), statePath)
	} else {
		fmt.Println("[Startup] No existing monitor state found, starting fresh.")
	}

	// ── Run ───────────────────────────────────────────────────────────────────
	urls := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
		"https://invalid.example",
	}

	start := time.Now()
	serviceTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), serviceTimeout)
	defer cancel()

	interval := 10 * time.Second

	paths := scheduler.Paths{State: statePath, Chain: chainPath}
	scheduler.Run(ctx, mon, chain, urls, interval, paths)

	// ── Final report ──────────────────────────────────────────────────────────
	fmt.Println()
	for url, status := range mon.Snapshot() {
		if status.LastError != nil {
			fmt.Printf("%s DOWN (%d consecutive failures)\n", url, status.ConsecutiveFailures)
		} else {
			fmt.Printf("%s UP %d\n", url, status.LastStatusCode)
		}
		if !status.LastChanged.IsZero() {
			fmt.Printf("  Changed at %v\n", status.LastChanged.Format(time.TimeOnly))
		}
	}

	fmt.Println("\n--- Ledger Verification ---")
	if chain.Validate() {
		fmt.Println("✅ SUCCESS: Ledger chain is mathematically valid and untampered.")
	} else {
		fmt.Fprintln(os.Stderr, "❌ CRITICAL ERROR: Ledger chain validation failed!")
		os.Exit(1)
	}

	tipHash, length := chain.TipHash()
	fmt.Printf("   Chain length: %d | Tip hash: %x\n", length, tipHash)
	fmt.Println("\nFinished in", time.Since(start))
}
