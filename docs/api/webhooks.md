# Webhooks

Subscribe to Dispatch events for real-time notifications about email delivery, subscriber changes, and compliance actions.

> 🚧 **Status:** Webhooks are planned for v0.2. This document describes the intended API.

---

## Register Webhook

```
POST /api/v1/sites/{site}/webhooks
```

### Request

```json
{
  "url": "https://mysite.com/hooks/email",
  "events": ["message.delivered", "message.bounced", "subscriber.unsubscribed"],
  "secret": "whsec_your_secret_key"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | ✅ | HTTPS URL to receive webhook payloads |
| `events` | array | ✅ | List of event types to subscribe to |
| `secret` | string | ✅ | Secret key for HMAC-SHA256 signature verification |

### Response (201 Created)

```json
{
  "id": "wh_abc123",
  "url": "https://mysite.com/hooks/email",
  "events": ["message.delivered", "message.bounced", "subscriber.unsubscribed"],
  "active": true,
  "created_at": "2026-03-14T22:50:00Z"
}
```

---

## Event Types

| Event | Fired When |
|-------|-----------|
| `message.queued` | Email enters the send queue |
| `message.sent` | Email handed to backend |
| `message.delivered` | Backend confirms delivery |
| `message.bounced` | Hard or soft bounce received |
| `message.complained` | Spam complaint received |
| `message.failed` | All retries exhausted |
| `subscriber.created` | New subscriber added |
| `subscriber.confirmed` | Double opt-in confirmed |
| `subscriber.unsubscribed` | Unsubscribe processed |
| `subscriber.deleted` | GDPR erasure completed |

---

## Webhook Payload

```json
{
  "event": "message.delivered",
  "timestamp": "2026-03-14T22:50:03Z",
  "site": "my-site",
  "data": {
    "message_id": "msg_a1b2c3d4e5f6",
    "to": "user@example.com",
    "template": "welcome",
    "backend": "resend",
    "backend_id": "re_xxxxx"
  }
}
```

---

## Signature Verification

Every webhook payload is signed with HMAC-SHA256 using your secret. The signature is included in the `X-Dispatch-Signature` header.

### Verification Example (Python)

```python
import hmac
import hashlib

def verify_webhook(payload: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

### Verification Example (Go)

```go
func verifyWebhook(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

---

## List Webhooks

```
GET /api/v1/sites/{site}/webhooks
```

## Delete Webhook

```
DELETE /api/v1/sites/{site}/webhooks/{id}
```

---

## Retry Policy

Failed webhook deliveries are retried up to 5 times with exponential backoff (1min, 5min, 30min, 2h, 12h). If all retries fail, the webhook is marked as inactive.
