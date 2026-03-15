# Backends

Backends are pluggable email delivery providers. Dispatch ships with built-in support for SMTP and Resend, with SES and Listmonk planned. You can also build custom backends.

---

## Overview

Every site in Dispatch is configured with a backend that handles the actual email delivery. The backend receives a fully rendered email (subject, HTML, text, headers) and delivers it to the recipient.

```
Your App → Dispatch API → [Render Template] → Backend → Recipient's Inbox
```

### Available Backends

| Backend | Status | Best For | Cost |
|---------|--------|----------|------|
| `smtp` | ✅ Ready | Any SMTP server, self-hosted mail | Varies |
| `resend` | ✅ Ready | Developer-friendly, quick setup | Free tier: 3k/month |
| `ses` | 🚧 Planned | High volume, cost efficiency | ~$0.10/1,000 emails |
| `listmonk` | 🚧 Planned | Existing Listmonk users | Free (self-hosted) |

---

## Choosing a Backend

### SMTP

**Use when:** You already have an SMTP server, or you're using a provider that offers SMTP access (Gmail, Outlook 365, Fastmail, Postfix, etc.).

**Pros:** Universal, works with any provider, no vendor lock-in.
**Cons:** Requires managing SMTP credentials, some providers rate-limit.

### Resend

**Use when:** You want the simplest setup with good deliverability and a generous free tier.

**Pros:** Modern API, excellent developer experience, good deliverability, easy DNS setup.
**Cons:** SaaS dependency, costs scale with volume.

### Amazon SES (planned)

**Use when:** You're sending high volumes and want the lowest cost.

**Pros:** Cheapest at scale ($0.10/1,000), AWS ecosystem integration, excellent deliverability.
**Cons:** More complex setup (AWS account, IAM, domain verification), sandbox mode for new accounts.

### Listmonk (planned)

**Use when:** You already run Listmonk and want Dispatch as an orchestration layer on top.

**Pros:** Leverages existing Listmonk infrastructure, no additional delivery costs.
**Cons:** Requires running Listmonk alongside Dispatch.

---

## Backend Configuration

Backends are configured per-site in `site.yaml`:

```yaml
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"
```

The `backend` field must match a registered backend name. The `backend_config` is a flat key-value map passed to the backend's factory function.

### SMTP Configuration

```yaml
backend: smtp
backend_config:
  host: smtp.example.com      # SMTP server hostname
  port: "587"                  # Port (as string): 25, 465 (TLS), 587 (STARTTLS)
  username: "user@example.com" # SMTP auth username (optional)
  password: "${SMTP_PASSWORD}" # SMTP auth password (optional)
  tls: "true"                  # Use implicit TLS (port 465). Default: false (STARTTLS)
```

See [SMTP Backend](../backends/smtp.md) for full details.

### Resend Configuration

```yaml
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"  # Required: Resend API key
```

See [Resend Backend](../backends/resend.md) for full details.

---

## The Backend Interface

All backends implement this Go interface:

```go
type Backend interface {
    Name() string
    Send(ctx context.Context, msg *OutgoingMessage) (*SendResult, error)
    BatchSend(ctx context.Context, msgs []*OutgoingMessage) (*BatchResult, error)
    Health(ctx context.Context) error
    MaxBatchSize() int
}
```

### `Name() string`

Returns the backend identifier (e.g., `"smtp"`, `"resend"`). Must match the name used in site configs.

### `Send(ctx, msg) (*SendResult, error)`

Delivers a single email. This is the primary method called by the queue worker.

**Input: `OutgoingMessage`**

```go
type OutgoingMessage struct {
    MessageID  string            // Dispatch's internal message ID
    From       string            // Sender email address
    FromName   string            // Sender display name
    ReplyTo    string            // Reply-to address
    To         string            // Recipient email address
    Subject    string            // Rendered subject line
    HTML       string            // Rendered HTML body
    Text       string            // Rendered plain text body
    Headers    map[string]string // Custom headers (List-Unsubscribe, etc.)
    Tags       []string          // Tags for categorization
    Metadata   map[string]string // Custom metadata
}
```

**Output: `SendResult`**

```go
type SendResult struct {
    BackendID  string  // Provider's message ID (for tracking)
    Status     string  // "sent" or "queued" (at provider level)
}
```

### `BatchSend(ctx, msgs) (*BatchResult, error)`

Delivers multiple emails in one call. Backends that support batch APIs (like Resend) can optimize this. The default implementation falls back to sequential `Send()` calls.

### `Health(ctx) error`

Checks backend connectivity. Called by the `/api/v1/health` endpoint and `dispatch doctor`.

### `MaxBatchSize() int`

Returns the maximum number of emails per batch. `0` means no batch support (sequential only).

---

## Backend Router

Dispatch uses an internal router to map sites to backends:

```
Site "my-store"  → Resend backend instance
Site "blog"      → Resend backend instance (same instance, shared)
Site "internal"  → SMTP backend instance
```

When multiple sites use the same backend type with the same config, Dispatch shares the backend instance. This saves connections and resources.

---

## Backend Health Monitoring

The health endpoint reports backend status:

```bash
curl http://localhost:8080/api/v1/health
```

```json
{
  "backends": {
    "resend": "ok",
    "smtp": "error: SMTP connection failed: dial tcp: connection refused"
  }
}
```

Each backend's `Health()` method is called:
- **SMTP:** Attempts a TCP connection to the SMTP server
- **Resend:** Makes a test API call to `/domains`
- **SES:** Calls `GetSendQuota` API

---

## Building a Custom Backend

See [Custom Backends](../backends/custom.md) for a complete guide. In brief:

1. Create a new file in `internal/backend/`
2. Implement the `Backend` interface
3. Register with `backend.Register("mybackend", factory)`
4. Use in site configs: `backend: mybackend`
