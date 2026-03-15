# Changelog

All notable changes to Dispatch are documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
Versioning: [Semantic Versioning](https://semver.org/)

---

## [Unreleased] — v0.1.0

### Added
- Core Go service with chi HTTP router
- SQLite database support with embedded schema migrations
- PostgreSQL support (driver: postgres in config)
- SMTP backend — direct email delivery to any SMTP server
- Resend backend — Resend API integration
- Pluggable backend interface with global registry
- Background send queue with configurable workers and retry backoff
- Database-level queue locking for multi-worker safety and crash recovery
- Template engine supporting single-file and directory-based templates
- Subject line extraction from Go template comments (`{{/* subject: ... */}}`)
- Shared and site-specific base template layouts
- Auto-generated plain text from HTML when no `.txt` file provided
- Multi-site configuration via YAML files and directory scanning
- Environment variable interpolation in all config files (`${VAR}`)
- Global suppression list checked before every send (at API time and queue time)
- Subscriber management with consent logging
- GDPR export and forget endpoints
- Unsubscribe page (public, no auth required)
- API authentication: master keys, site-scoped keys, read-only keys
- Structured JSON logging via `log/slog`
- Health endpoint (`GET /api/v1/health`) with queue and backend status
- CLI: `dispatch serve`, `init`, `site add/list/remove`, `migrate`, `version`
- Docker and Docker Compose deployment
- Full documentation suite (getting-started, concepts, API reference, compliance, backends, SDKs, operations, development)
- GoDoc package comments on all internal packages

### Architecture
- Single binary, no external runtime dependencies
- SQLite WAL mode enabled by default for concurrent read performance
- Config loaded at startup, sites discovered by directory scan
- Template files read from disk on each render (no in-memory cache in v0.1)

---

## Roadmap

See [Release Process](docs/development/releases.md) for the full roadmap.

### v0.2.0 (Planned)
- Amazon SES backend
- Listmonk backend
- Webhook delivery system
- Double opt-in flow (confirmation tokens)
- Database-managed API keys (hashed, revocable)
- `dispatch doctor` — full diagnostics
- `dispatch keys` — full CLI key management
- `dispatch send` — CLI test send

### v0.3.0 (Planned)
- Prometheus metrics endpoint
- Rate limiting per site/backend
- Message log pagination and search
- Per-site suppression (optional)

### v1.0.0 (Planned)
- Optional web admin UI
- Security audit
- Stable API guarantee
