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
