# SchemaPing

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/rubensantoniorosa2704/schemaping-worker/actions/workflows/ci.yml/badge.svg)](https://github.com/rubensantoniorosa2704/schemaping-worker/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Detect API schema drift before your integrations break.

SchemaPing monitors HTTP JSON endpoints, compares response structures over time, and prints diffs in the terminal when something changes.

---

## What it detects

- Added or removed fields
- Type changes (`string` → `number`, etc.)
- Nullability changes
- Unexpected HTTP status codes
- Request failures and timeouts

---

## Getting started

**Prerequisites:** Go 1.25+

```bash
git clone https://github.com/rubensantoniorosa2704/schemaping-worker.git
cd schemaping-worker
go build -o schemaping ./cmd/schemaping
```

---

## Usage

```bash
# run a single check for all monitors and exit
schemaping check --config ./examples/config.yaml

# run continuously, checking on each monitor's interval
schemaping run --config ./examples/config.yaml

# override the interval for all monitors
schemaping run --config ./examples/config.yaml --interval 30s
```

---

## Configuration

```yaml
monitors:
  - name: payments-api
    url: https://api.example.com/v1/payments
    method: GET
    interval: 5m
    timeout: 10s
    expected_status: 200
    headers:
      Authorization: Bearer YOUR_TOKEN
```

| Field | Default | Description |
|---|---|---|
| `name` | required | Unique monitor identifier |
| `url` | required | Endpoint to monitor |
| `method` | `GET` | HTTP method |
| `interval` | `1m` | How often to check |
| `timeout` | `10s` | Request timeout |
| `expected_status` | `200` | Expected HTTP status code |
| `headers` | — | Optional request headers |

---

## Terminal output

```
[payments-api] change detected
  + customer.phone added (string)
  - customer.document removed (string)
  ~ amount changed: string -> number
  ~ status changed: 200 -> 404
```

Snapshots are saved to `~/.schemaping/snapshots/` after each check.

---

## Roadmap

- [ ] Webhook alerts
- [ ] Postgres persistence
- [ ] Snapshot history
- [ ] OpenAPI diff support

---

## License

Apache License 2.0 — see [LICENSE](./LICENSE).
