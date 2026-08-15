# Testing Guide

This guide covers how to run and extend the test suite for the server package.

## Running tests

```bash
# Full suite
go test ./... -count=1

# A single package
go test ./pkg/server/... -v

# With race detection
go test ./pkg/server/... -race -count=1
```

## Test categories

### Unit tests

Unit tests cover individual functions in isolation. The configuration switch
logic and middleware chain construction are the highest-value targets because
they contain the concurrency-sensitive code.

### Integration tests

Integration tests exercise the server with a real configuration file and a
real listener, asserting that a reload swaps the entrypoint handler and that
requests are served by the new handler.

### Race tests

The watcher/switch path is exercised concurrently with request serving, so it
must pass `-race`. Run race tests in CI on every pull request.

## Key behaviors to test

### First reload

The first configuration reload is the most error-prone path. Historically the
entrypoint seed value and the swapped value used different concrete types,
causing a panic on the first `Store`. A test must assert that:

1. The server starts and serves the initial configuration.
2. A configuration change triggers exactly one swap.
3. No panic occurs during the first swap.
4. Subsequent requests are served by the new handler.

### Panic containment

A test must assert that a malformed configuration update does not kill the
watcher goroutine: the server should log the error and continue serving the
previous configuration.

### Handler correctness

After a swap, a request to a route that changed must hit the new handler, and
a route that did not change must still hit the original handler.

## Writing tests

- Prefer table-driven tests for the switch logic.
- Use a test double for the configuration source so reloads are deterministic.
- Keep assertions on observable behavior (which handler served the request),
  not on internal fields.

## CI

The `quality.yml` workflow runs `go vet`, `go build`, and `go test` on every
pull request. A PR that fails any of these is blocked from merge.
