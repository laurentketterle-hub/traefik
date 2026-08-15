# Server Architecture

This document describes the structure of the server package and the lifecycle
of its dynamic configuration handling.

## Package layout

```
pkg/
  server/     configuration model, entrypoint management, and hot-reload watcher
  middleware/ middleware chain construction
  config/     static configuration parsing
```

## Dynamic configuration model

The server is designed around a single, mutable configuration handle that can
be swapped at runtime. The core components are:

1. **Entrypoint registry** — maps a named entrypoint to the handler that serves
   it. This mapping is the object that changes during a configuration reload.
2. **Configuration watcher** — observes the configuration source for updates
   and triggers a swap when a new configuration is detected.
3. **Configuration switch** — atomically publishes the new entrypoint mapping
   so that in-flight requests are not served a half-built state.

## Hot-reload flow

```
config source changes
        |
        v
watcher detects update
        |
        v
new entrypoint mapping built
        |
        v
atomic swap of the entrypoint handler
        |
        v
subsequent requests served by new mapping
```

## Correctness constraints

- **Type-consistent atomic store.** The entrypoint handle is published through
  `sync/atomic`. Every store must use the exact same concrete type; mixing an
  `http.HandlerFunc` seed value with a `*http.ServeMux` on a later store panics
  with `store of inconsistently typed value`. The seed value must be normalized
  to the final concrete type before the first store.
- **Panic containment.** The watcher goroutine must not be allowed to die on a
  reload error, otherwise future configuration updates are silently ignored.
  Any code path that can panic during a swap must be recovered and surfaced as
  a logged error.
- **No partial publication.** A new mapping must be fully constructed before it
  is published; publishing a partially-wired mapping exposes requests to stale
  or nil handlers.

## Testing guidance

- Unit tests for the switch path should exercise the first reload specifically,
  since that is where the seed-value type mismatch historically surfaces.
- Race detection (`go test -race`) is recommended for the watcher/switch path
  because it is exercised concurrently with request serving.
