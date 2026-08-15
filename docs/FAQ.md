# Frequently Asked Questions

## Configuration

### How do I reload the configuration without restarting?

Update the configuration file. The watcher detects the change and swaps the
entrypoint mapping atomically.

### What happens if my new configuration is invalid?

The server logs the error and keeps serving the previous configuration. Fix
the file and save it again.

### How do I add a new route?

Add an entry under `routes` with a unique `name`, a `path`, and a `handler`.
Restart not required; save the file to reload.

## Middleware

### In what order does middleware run?

Middleware runs in the order listed in the route's `middleware` array.

### Can I reuse middleware across routes?

Yes. Middleware is named and referenced by name in each route.

## Troubleshooting

### The server panicked on the first configuration update

This indicates a mixed-type `atomic.Value` store. Ensure the seed entrypoint
value is normalized to the final concrete type before the first store.

### The watcher stopped applying updates

The watcher goroutine may have died on an unhandled panic. Ensure panic
recovery is in place on the reload path.

### Requests hit stale handlers after a reload

Confirm the new mapping is fully built before it is published; a partially
wired mapping can leave some routes on old handlers.
