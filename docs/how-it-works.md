# How SchemaPing Works

**Author:** Rubens Antonio Rosa  
**Date:** 2026-04-21  
**Version:** 0.1.0

---

## Table of Contents

- [Overview](#overview)
- [Execution Flow](#execution-flow)
- [Package Responsibilities](#package-responsibilities)
- [Change Detection](#change-detection)
- [Snapshot Format](#snapshot-format)
- [CLI Commands](#cli-commands)

---

## Overview

SchemaPing runs HTTP checks against configured endpoints, saves a snapshot of each response, and compares it against the previous snapshot to detect structural changes in the JSON body or HTTP status code.

It does not monitor uptime. It monitors **contract**.

---

## Execution Flow

```
config.Load()
    └── for each Monitor
            └── checker.Run()
                    ├── httpclient.Do()       → HTTP request
                    └── Snapshot{}            → status + parsed body

            └── storage/file.Load()           → previous Snapshot from disk
            └── storage/file.Save(current)    → overwrite snapshot on disk
            └── diff.Compare(prev, current)   → []DiffResult
            └── print results to terminal
```

In `run` mode, the scheduler fires this flow for each monitor on its configured interval, in a separate goroutine per monitor. On `SIGINT` or `SIGTERM`, all goroutines stop gracefully.

---

## Package Responsibilities

| Package | File | Responsibility |
|---|---|---|
| `pkg/types` | [`pkg/types/types.go`](../pkg/types/types.go) | Shared domain types: `Monitor`, `Snapshot`, `DiffResult`, `ChangeKind` |
| `internal/config` | [`internal/config/config.go`](../internal/config/config.go) | Load and validate `config.yaml`, apply defaults |
| `internal/httpclient` | [`internal/httpclient/httpclient.go`](../internal/httpclient/httpclient.go) | Execute HTTP request with timeout and headers |
| `internal/checker` | [`internal/checker/checker.go`](../internal/checker/checker.go) | Orchestrate a single check, build `Snapshot`, classify failures |
| `internal/diff` | [`internal/diff/diff.go`](../internal/diff/diff.go) | Compare two snapshots, return list of structural changes |
| `internal/scheduler` | [`internal/scheduler/scheduler.go`](../internal/scheduler/scheduler.go) | Run monitors on intervals, handle OS signals for graceful stop |
| `internal/storage/file` | [`internal/storage/file/file.go`](../internal/storage/file/file.go) | Persist and load snapshots as JSON files in `~/.schemaping/snapshots/` |
| `cmd/schemaping` | [`cmd/schemaping/main.go`](../cmd/schemaping/main.go) | CLI entrypoint, flag parsing, terminal output |

---

## Change Detection

`diff.Compare(before, after Snapshot)` walks both JSON bodies recursively using dot-notation paths and returns a `[]DiffResult`.

| Change Kind | Condition | Example output |
|---|---|---|
| `added` | Key present in `after`, missing in `before` | `+ customer.phone added (string)` |
| `removed` | Key present in `before`, missing in `after` | `- customer.document removed (string)` |
| `type_changed` | Key present in both, but JSON type differs | `~ amount changed: string -> number` |
| `nullability_changed` | One side is `null`, the other is not | `~ user.address nullability changed: null -> object` |
| `status_changed` | HTTP status code differs between snapshots | `~ status changed: 200 -> 404` |

Arrays are compared by type only — their contents are not diffed.

---

## Snapshot Format

Each snapshot is stored as a JSON file at `~/.schemaping/snapshots/<monitor-name>.json`.  
Only the latest snapshot is kept per monitor — there is no history yet.

```json
{
  "monitor_name": "payments-api",
  "captured_at": "2026-04-21T19:00:00Z",
  "status_code": 200,
  "body": {
    "id": 1,
    "status": "active"
  }
}
```

If a request fails, `error` is set and `body` is omitted:

```json
{
  "monitor_name": "payments-api",
  "captured_at": "2026-04-21T19:00:00Z",
  "status_code": 0,
  "error": "httpclient: execute request: context deadline exceeded"
}
```

---

## CLI Commands

| Command | Description |
|---|---|
| `schemaping check --config <path>` | Run one check for all monitors and exit |
| `schemaping run --config <path>` | Run continuously on each monitor's interval |
| `schemaping run --config <path> --interval <duration>` | Override interval for all monitors (e.g. `30s`, `2m`) |
| `schemaping --help` | Show usage |
| `schemaping --version` | Show version |
