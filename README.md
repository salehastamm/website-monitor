# website-monitor

A concurrent website monitoring service built in Go as a learning project. It
progressed through eight phases, each introducing a new Go concept on top of
the last.

## Features

- **Concurrent checking** — a worker pool fans out HTTP GETs across all URLs simultaneously
- **Per-request timeouts** — each check gets a 5-second deadline via `context.WithTimeout`
- **Exponential-backoff retries** — transient failures (network errors, 5xx) are retried up to 3 times; `context.Canceled` is never retried
- **Periodic scheduling** — a `time.Ticker` re-checks all sites on a configurable interval
- **Content-change detection** — the SHA-256 of each response body is compared round-to-round; changes are flagged and timestamped
- **Merkle tree** — each round's body hashes are committed into a Merkle tree; the root summarises the entire round in a single hash, and inclusion proofs can verify individual sites
- **Hash-chained ledger** — Merkle roots are chained into an append-only blockchain-style ledger; `Validate()` walks the full chain to detect any tampering
- **Persistent storage** — the ledger and monitor state are written to disk after every round and reloaded on the next startup, so history survives restarts

## Layout

```
cmd/monitor/           # package main — wires everything together and drives the run loop
internal/
  checker/             # CheckWebsite (timed HTTP GET + body hash), CheckAll (worker pool), CheckWithRetry
  monitor/             # thread-safe status map: last code, consecutive failures, last hash, change timestamp
  scheduler/           # Run: immediate first check then periodic ticks; calls store after each round
  merkle/              # Root, Proof, Verify — domain-separated leaf/node hashing
  ledger/              # append-only hash chain of Merkle roots with Validate()
  store/               # SaveState / LoadState (atomic JSON), AppendBlock / LoadChain (JSON Lines)
```

## Persistent files

Two files are written to the working directory at runtime:

| File | Format | Updated |
|---|---|---|
| `monitor_state.json` | Pretty-printed JSON object | Atomically replaced after every round |
| `ledger.jsonl` | JSON Lines (one block per line) | Appended after every round |

On startup the program loads both files, validates the full chain, and continues
appending — `consecutive_failures` and content-change timestamps carry over across
restarts.

## Run

```sh
go run ./cmd/monitor
```

Example output (first run — no existing files):

```
[Startup] No existing ledger found, starting a fresh chain.
[Startup] No existing monitor state found, starting fresh.
[Scheduler] Round observed 3 sites. Merkle root: be3ae15d…
[Ledger] Chain Length: 1 | Tip Hash: 98c3c304…
...
https://invalid.example DOWN (3 consecutive failures)
https://example.com UP 200
https://github.com UP 200
  Changed at 13:10:27
https://google.com UP 200
  Changed at 13:10:27

--- Ledger Verification ---
✅ SUCCESS: Ledger chain is mathematically valid and untampered.
   Chain length: 3 | Tip hash: 9a75cef1…

Finished in 30.003s
```

Subsequent runs load and resume from disk:

```
[Startup] Loaded 3 blocks from ledger.jsonl — chain is valid ✅
[Startup] Restored prior status for 4 site(s) from monitor_state.json
[Scheduler] Round observed 3 sites. Merkle root: 2e303ffc…
[Ledger] Chain Length: 4 | Tip Hash: 654fe36e…
```

## Test

```sh
go test ./...
```

All tests use `net/http/httptest` and `os.TempDir` — they run instantly and
never touch the real network or leave files on disk.

## Phase history

| Phase | Concept introduced |
|---|---|
| 1 | Sequential URL checks |
| 2 | Channel-based concurrency (one goroutine per URL) |
| 3 | Worker pool with backpressure (bounded goroutines, unbuffered job channel) |
| 4 | Shared state — mutex-protected `Monitor` status map |
| 5 | Context propagation — per-request timeouts, cooperative cancellation |
| 6 | Periodic scheduling via `time.Ticker` |
| 7 | Exponential-backoff retries for transient failures |
| 8 | Content-change detection (SHA-256 body hash), Merkle tree, hash-chained ledger, persistent storage |
