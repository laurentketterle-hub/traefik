# Configuration Reference

This page documents the server's configuration surface.

## Top-level structure

```yaml
entrypoints:
  http:
    address: ":8080"
routes:
  - name: health
    path: /health
    handler: health
  - name: proxy
    path: /
    handler: proxy
```

## Entrypoints

An entrypoint defines a listener. Supported fields:

| Field | Type | Description |
|---|---|---|
| `address` | string | Listen address (`:8080`) |
| `tls` | object | Optional TLS settings |
| `timeout` | duration | Read/write timeout |

## Routes

A route maps a path to a handler plus an optional middleware chain.

| Field | Type | Description |
|---|---|---|
| `name` | string | Unique route name |
| `path` | string | URL path to match |
| `handler` | string | Named handler to invoke |
| `middleware` | list | Ordered middleware names |

## Middleware

Middleware wraps a handler and runs in the order listed. Common middleware
includes logging, authentication, and rate limiting.

## Reload semantics

- The server watches the configuration file for changes.
- On a valid change, the entrypoint mapping is swapped atomically.
- On an invalid change, the error is logged and the previous configuration
  continues to serve.

## Health check

The server exposes a health endpoint. Configure it as:

```yaml
health:
  path: /health
```

The health endpoint returns `200` while the server is serving.

## Example

```yaml
entrypoints:
  http:
    address: ":8080"
health:
  path: /health
routes:
  - name: health
    path: /health
    handler: health
  - name: app
    path: /
    handler: app
    middleware:
      - logging
      - auth
```
