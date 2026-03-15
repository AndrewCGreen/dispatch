# Release Process

How Dispatch versions are managed and released.

---

## Versioning

Dispatch follows [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`

| Type | When | Example |
|------|------|---------|
| PATCH | Bug fixes, no API changes | `0.1.0` → `0.1.1` |
| MINOR | New features, backwards compatible | `0.1.0` → `0.2.0` |
| MAJOR | Breaking changes | `0.x.0` → `1.0.0` |

During the `0.x` phase, minor versions may contain breaking changes. We'll document them clearly in CHANGELOG.md.

---

## Roadmap

### v0.1.0 — Foundation (Current)
- ✅ Core Go service
- ✅ SMTP backend
- ✅ Resend backend
- ✅ SQLite store with migrations
- ✅ Send queue with retries
- ✅ Template engine
- ✅ Basic auth (config-based keys)
- ✅ Subscriber management
- ✅ Global suppression list
- ✅ GDPR export/forget endpoints
- ✅ CLI (serve, init, site add/list/remove)
- ✅ Docker deployment
- ✅ Full documentation

### v0.2.0 — Backends & Events
- SES backend
- Listmonk backend
- Webhook delivery system
- Double opt-in flow (complete)
- Database-managed API keys
- Batch send optimizations
- `dispatch doctor` full implementation
- `dispatch keys` CLI implementation
- `dispatch send` test send CLI

### v0.3.0 — Compliance & Scale
- PostgreSQL migration support
- Prometheus metrics endpoint
- Per-site suppression (optional)
- Unsubscribe token verification
- Consent log querying API
- Rate limiting per site/backend
- Message log pagination

### v1.0.0 — Production Ready
- Optional web admin UI
- Security audit
- Performance benchmarks
- Horizontal scaling documentation
- Community backends documented
- Comprehensive test coverage (>80%)
- Stable API (no breaking changes after this)

---

## CHANGELOG

### Unreleased

- Initial release

---

## Release Steps (Maintainers)

```bash
# 1. Update version in cmd/root.go
const version = "0.2.0"

# 2. Update CHANGELOG.md

# 3. Commit
git add -A
git commit -m "chore: release v0.2.0"

# 4. Tag
git tag -a v0.2.0 -m "v0.2.0"
git push origin main v0.2.0

# 5. GitHub Actions builds and publishes:
#    - Binaries (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64)
#    - Docker image (ghcr.io/dispatch-email/dispatch:0.2.0 and :latest)
#    - GitHub Release with changelog
```
