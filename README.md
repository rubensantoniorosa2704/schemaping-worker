# SchemaPing

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/rubensantoniorosa2704/schemaping-worker/actions/workflows/ci.yml/badge.svg)](https://github.com/rubensantoniorosa2704/schemaping-worker/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Detect API schema drift before your integrations break.

SchemaPing monitors HTTP JSON endpoints, compares response structures over time, and alerts you when something changes — in the terminal and via webhooks.

---

## What it detects

- Added or removed fields
- Type changes (`string` → `number`, etc.)
- Nullability changes
- Unexpected HTTP status codes
- Request failures and timeouts

> **Array diffing limitation:** arrays are compared using only their first element as a schema representative. Heterogeneous arrays, nested arrays, and length changes are not reported.

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

# verify that all configured webhooks are reachable
schemaping test-webhooks --config ./examples/config.yaml
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
    retries: 3        # retry up to 3 times on transient failures (5xx, 429, timeouts)
    retry_backoff: 2s # base interval for exponential backoff (2s → 4s → 8s…, capped at 30s)
    headers:
      Authorization: Bearer ${API_TOKEN}

webhooks:
  - type: discord
    url: ${DISCORD_WEBHOOK_URL}
```

| Field | Default | Description |
|---|---|---|
| `name` | required | Unique monitor identifier |
| `url` | required | Endpoint to monitor |
| `method` | `GET` | HTTP method |
| `interval` | `1m` | How often to check |
| `timeout` | `10s` | Request timeout |
| `expected_status` | `200` | Expected HTTP status code |
| `retries` | `3` | Additional attempts after a transient failure (5xx, 429, timeout). Set to `0` to disable. |
| `retry_backoff` | `2s` | Base duration for exponential backoff between retries (capped at 30s). |
| `headers` | — | Optional request headers |
| `webhooks` | — | Per-monitor webhook override (see below) |

---

## Webhook alerts

SchemaPing can send notifications to external platforms when a schema change is detected.

### Supported platforms

| Platform | `type` | Required fields |
|---|---|---|
| Discord | `discord` | `url` |
| Telegram | `telegram` | `url`, `chat_id` |

### Global webhooks

Defined once at the top level, they fire for every monitor that detects a change:

```yaml
webhooks:
  - type: discord
    url: ${DISCORD_WEBHOOK_URL}
  - type: telegram
    url: https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage
    chat_id: ${TELEGRAM_CHAT_ID}

monitors:
  - name: payments-api
    url: https://api.example.com/v1/payments
  - name: users-api
    url: https://api.example.com/v1/users
```

### Per-monitor override

A monitor can define its own `webhooks` list, which replaces the global list for that monitor only:

```yaml
webhooks:
  - type: discord
    url: ${DISCORD_WEBHOOK_URL}   # default channel

monitors:
  - name: payments-api
    url: https://api.example.com/v1/payments
    # no webhooks field → uses global

  - name: critical-api
    url: https://api.example.com/critical
    webhooks:                     # override → different channel
      - type: discord
        url: ${DISCORD_CRITICAL_WEBHOOK_URL}

  - name: noisy-api
    url: https://api.example.com/noisy
    webhooks: []                  # silenced → no notifications
```

### Environment variables

Tokens and webhook URLs should never be hardcoded in the config file. Use `${ENV_VAR}` references — they are expanded at startup before the YAML is parsed, so they work in any field.

```bash
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
export TELEGRAM_BOT_TOKEN="your-bot-token"
export TELEGRAM_CHAT_ID="your-chat-id"

schemaping run --config ./config.yaml
```

### Discord setup

1. Open the target channel → **Edit Channel** → **Integrations** → **Webhooks** → **New Webhook**
2. Give it a name and copy the webhook URL
3. Export the URL: `export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."`

### Telegram setup

1. Create a bot via [@BotFather](https://t.me/BotFather) and copy the token
2. Get your chat ID (send a message to the bot and call `getUpdates`)
3. Export both: `export TELEGRAM_BOT_TOKEN="..."` and `export TELEGRAM_CHAT_ID="..."`

### Verifying your setup

```bash
schemaping test-webhooks --config ./config.yaml
```

```
Testing 2 webhook(s)...
  [1] OK
  [2] OK
```

Exits with code `1` if any webhook fails, making it safe to use in CI/CD pipelines.

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

- [x] Webhook alerts (Discord, Telegram)
- [x] Retry with exponential backoff
- [ ] Postgres persistence
- [ ] Snapshot history
- [ ] OpenAPI diff support

---

## License

Apache License 2.0 — see [LICENSE](./LICENSE).
