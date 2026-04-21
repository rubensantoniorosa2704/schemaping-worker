# Contributing to SchemaPing

Thanks for your interest in contributing.

---

## Opening issues

Use issues to report bugs, propose features, or share real-world API drift cases.

### Bug report

**Title:** `[bug] short description of the problem`

Include:
- What you ran (command + config)
- What you expected to happen
- What actually happened
- Go version and OS

### Feature request

**Title:** `[feature] short description`

Include:
- The problem you're trying to solve
- Why it belongs in the open-source core (not a hosted product feature)
- Any relevant examples

### API drift case

**Title:** `[drift-case] provider name — what changed`

Share real examples of API changes that broke (or almost broke) an integration. These help shape what SchemaPing should detect.

---

## Pull requests

- Open an issue before starting significant work
- Keep changes focused — one concern per PR
- Match the existing code style (idiomatic Go, minimal abstractions)
- Do not add dependencies without discussion

---

## Scope

Before contributing, check that your idea fits the project scope:

- HTTP JSON monitoring only (for now)
- Terminal-first experience
- No dashboard, no auth, no billing
- Simple self-hosting

If unsure, open an issue first.
