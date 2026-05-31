# Veritect

**Enterprise Database Schema Drift Engine**

Veritect is a headless, production-grade Go application designed to detect database schema drift in CI/CD pipelines. It computes a deterministic SHA-256 fingerprint of your PostgreSQL schema and fails the build when an unexpected structural change is introduced.

---

## Why Veritect

In fast-moving engineering teams, schema changes can slip into production unnoticed. Veritect treats your database schema as versioned infrastructure: it snapshots the canonical structure, stores a baseline in `veritect.lock`, and raises a hard failure on any drift. Zero cost, zero runtime overhead, zero surprises.

---

## Quick Start

```bash
# 1. Set environment variables
export DATABASE_URL="postgres://user:pass@localhost:5432/dbname"
export SLACK_WEBHOOK="https://hooks.slack.com/services/..."  # optional

# 2. Initialize baseline (creates veritect.lock)
go run ./cmd/veritect

# 3. On subsequent runs, any schema drift exits with code 1
go run ./cmd/veritect
```

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string (libpq format) |
| `SLACK_WEBHOOK` | No | Slack Incoming Webhook URL for drift alerts |

---

## CI Integration

Add Veritect to any GitHub Actions pipeline as a zero-dependency step:

```yaml
- name: Check Schema Drift
  run: go run ./cmd/veritect
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
    SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK }}
```

The action will:
1. **Pass (exit 0)** on the first run, auto-creating `veritect.lock`.
2. **Pass (exit 0)** when the schema matches the locked baseline.
3. **Fail (exit 1)** on drift, printing the structural diff and optionally notifying Slack.

---

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   cmd/veritect  │────▶│  internal/       │────▶│  information_   │
│   main.go       │     │  database/       │     │  schema.columns │
│                 │     │  postgres.go     │     │  (public schema)│
│                 │     └──────────────────┘     └─────────────────┘
│                 │                │
│                 │                ▼
│                 │     ┌──────────────────┐
│                 │────▶│  internal/       │────▶ Slack Webhook
│                 │     │  notifier/       │
│                 │     │  slack.go        │
└─────────────────┘     └──────────────────┘
```

- **Headless**: No HTTP server, no background processes.
- **Modular**: Clean separation between database access, notification, and orchestration.
- **Deterministic**: Queries enforce strict ordering before hashing to eliminate non-determinism.
- **Zero-cost**: Only runs when invoked; no persistent services or infrastructure required.

---

## License

Licensed under the Business Source License 1.1. See [LICENSE](./LICENSE) for details.

- **Change Date**: 2030-05-31
- **Change License**: Apache-2.0

Internal and non-commercial use is unrestricted. Competing commercial SaaS offerings require a separate license.
