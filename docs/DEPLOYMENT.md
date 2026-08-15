# Deployment & Operations Guide

This document describes how the server is built, configured, and operated in
production.

## Build

```bash
go build -o server ./cmd/server
```

Cross-compile for a target platform with standard Go flags:

```bash
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
```

## Configuration

The server reads its configuration from a file and watches it for changes.
Configuration covers:

- **Entrypoints** — named listener definitions (address, port, TLS settings).
- **Routes** — path-to-handler mappings, including the middleware chain.

A configuration reload is applied without restarting the process: the watcher
detects the change and atomically swaps the entrypoint mapping.

## Running

```bash
./server --config ./config.yaml
```

For containerized deployments, mount the configuration file read-only and
signal reloads by updating the mounted file.

## Configuration reload semantics

- A valid new configuration is applied atomically.
- An invalid configuration is rejected with an error log; the previous
  configuration keeps serving.
- A reload must never drop in-flight requests.

## Health checks

Expose a health endpoint that returns `200` when the server is serving. Use it
for liveness and readiness probes in orchestrators.

## Logging

- Log configuration loads and reloads with the resulting entrypoint count.
- Log any reload rejection with the parse error so operators can fix it.
- Avoid logging request bodies.

## Common issues

| Symptom | Cause | Action |
|---|---|---|
| Panic on first reload | Mixed concrete types in `atomic.Value` | Normalize seed value to final type |
| Watcher stops after error | Unrecovered panic in watcher | Add recovery + error logging |
| Stale routes after reload | Incomplete publication | Build full mapping before swap |
