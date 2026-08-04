# Configuration Reference

**Author:** Rubens Antonio Rosa  
**Date:** 2026-04-21  
**Version:** 0.1.0

---

## Table of Contents

- [File location](#file-location)
- [Structure](#structure)
- [Fields](#fields)
- [Examples](#examples)

---

## File location

SchemaPing loads the config file from the path passed via `--config`:

```bash
schemaping run --config ./config.yaml
schemaping check --config /etc/schemaping/config.yaml
```

If `--config` is omitted, SchemaPing looks for `./config.yaml` in the current directory.

> Do not commit config files that contain credentials. Add `config.yaml` to your `.gitignore`.  
> Use [`examples/config.yaml`](../examples/config.yaml) as a safe starting point.

---

## Structure

```yaml
monitors:
  - name: <string>
    url: <string>
    method: <string>
    interval: <duration>
    timeout: <duration>
    expected_status: <int>
    retries: <int>
    retry_backoff: <duration>
    headers:
      <key>: <value>
```

---

## Fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `name` | string | yes | — | Unique identifier for the monitor. Used in terminal output and snapshot filenames. |
| `url` | string | yes | — | Full URL of the endpoint to monitor. |
| `method` | string | no | `GET` | HTTP method (`GET`, `POST`, etc.). |
| `interval` | duration | no | `1m` | How often to run the check in `run` mode. Accepts Go duration strings: `30s`, `5m`, `1h`. |
| `timeout` | duration | no | `10s` | Request timeout. If exceeded, the check is recorded as an error. |
| `expected_status` | int | no | `200` | Expected HTTP status code. A different status is recorded as an error in the snapshot. |
| `retries` | int | no | `3` | Number of additional attempts after a transient failure (5xx, 429, or timeout). Set to `0` to disable retries entirely. |
| `retry_backoff` | duration | no | `2s` | Base duration for exponential backoff between retries. Progression: `base × 2^(attempt−1)`, capped at 30s. E.g. with `2s`: 2s → 4s → 8s → 16s → 30s. |
| `headers` | map | no | — | HTTP headers sent with every request. Use for authentication or content negotiation. |

---

## Examples

### Minimal

```yaml
monitors:
  - name: users-api
    url: https://api.example.com/v1/users
```

### With authentication

```yaml
monitors:
  - name: payments-api
    url: https://api.example.com/v1/payments
    headers:
      Authorization: Bearer YOUR_TOKEN
```

### API key in header

```yaml
monitors:
  - name: weather-api
    url: https://api.weather.com/current
    headers:
      X-API-Key: YOUR_KEY
```

### Custom interval and timeout

```yaml
monitors:
  - name: slow-api
    url: https://api.example.com/report
    interval: 30m
    timeout: 30s
```

### Multiple monitors

```yaml
monitors:
  - name: users-api
    url: https://api.example.com/v1/users
    interval: 1m

  - name: orders-api
    url: https://api.example.com/v1/orders
    interval: 5m
    headers:
      Authorization: Bearer YOUR_TOKEN

  - name: internal-service
    url: https://internal.example.com/status
    expected_status: 204
    interval: 30s
```

---

## Retry and backoff

By default, SchemaPing retries up to **3 times** when a transient failure is detected. A failure is considered transient when:

- A transport error occurs (timeout, connection refused, DNS failure)
- The server returns HTTP **429** (rate limited)
- The server returns HTTP **5xx** (server error)

Non-transient failures (e.g. HTTP 404, JSON parse errors) are **not retried** — they are reported immediately, as they likely indicate a contract change.

### Backoff formula

The wait before each retry follows an exponential curve:

```
wait = retry_backoff × 2^(attempt − 1)   (max 30s)
```

With the default `retry_backoff: 2s`:

| Attempt | Wait before retry |
|---------|-------------------|
| 1st retry | 2s |
| 2nd retry | 4s |
| 3rd retry | 8s |

### Disabling retries

Set `retries: 0` to report the first failure immediately:

```yaml
monitors:
  - name: strict-api
    url: https://api.example.com/strict
    retries: 0
```

### Custom retry settings

```yaml
monitors:
  - name: flaky-api
    url: https://flaky.example.com/api
    retries: 5
    retry_backoff: 1s   # 1s → 2s → 4s → 8s → 16s
```

### Terminal output during retries

When a retry happens, SchemaPing logs the attempt to stderr:

```
[payments-api] retrying (attempt 1/4, reason: httpclient: execute request: context deadline exceeded, next in 2s)
```
