# Listmonk Backend

> 🚧 **Status:** Planned for v0.2. This document describes the intended implementation.

The Listmonk backend routes email delivery through an existing [Listmonk](https://listmonk.app) instance. This is for users who already run Listmonk and want to use Dispatch as an orchestration layer on top.

---

## Why Use Dispatch + Listmonk Together?

Dispatch adds value on top of Listmonk:
- **Multi-site management** — one Dispatch instance routes to multiple Listmonk instances or multiple lists
- **Unified API** — your applications call Dispatch's consistent API; Listmonk is the delivery engine
- **SDK support** — Python and Go SDKs for Dispatch abstract Listmonk's API
- **GDPR orchestration** — Dispatch's forget endpoint handles cross-site erasure

---

## Planned Configuration

```yaml
backend: listmonk
backend_config:
  url: "https://listmonk.yourdomain.com"
  username: "${LISTMONK_USERNAME}"
  password: "${LISTMONK_PASSWORD}"
  list_id: "3"          # Listmonk list ID for this site
```

---

## How It Works

When Dispatch sends an email via the Listmonk backend:
1. Dispatch renders the template (subject, HTML, text)
2. Calls Listmonk's transactional email API: `POST /api/tx`
3. Listmonk handles the actual SMTP delivery using its own configured SMTP backend

This means you still need an SMTP backend configured in Listmonk. The Listmonk backend is a routing layer, not a delivery mechanism.

---

## Alternative: Use Listmonk's SMTP

If you want Listmonk's SMTP settings but don't need Listmonk's UI or list management, you can use the SMTP backend directly with the same SMTP credentials that Listmonk uses. This is simpler than the Listmonk backend.
