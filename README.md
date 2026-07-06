# website-monitor

A concurrent website monitoring service, built in Go as a learning project.

**Phase 1** is a command-line tool that checks a list of websites **concurrently**
and reports, for each one, whether it was reachable, its HTTP status code, and how
long the request took.

## Layout

```
cmd/monitor/       # package main — the CLI entry point (parses input, prints results)
internal/checker/  # the reusable logic: CheckWebsite performs a timed HTTP GET
```

The `checker` package does the real work and is independently unit-tested; `main`
is a thin shell that launches one goroutine per URL and prints the collected results.

## Run

```sh
go run ./cmd/monitor
```

Example output:

```
https://example.com returned 200 in 1.70s
https://google.com returned 200 in 2.76s
https://github.com returned 200 in 1.71s
https://invalid.example failed : Get "https://invalid.example": EOF in 9.50s
Monitoring Websites Concurrently 9.50s
```

Each site is checked in its own goroutine, so the total time is roughly the
**slowest single request** rather than the sum of them all. Note that a single
slow or unreachable host currently bounds the total run time — per-request
timeouts (a later phase) will fix that.

## Test

```sh
go test ./...
```

The tests use `net/http/httptest` to spin up a local fake server, so they run
instantly and never depend on the real internet.
