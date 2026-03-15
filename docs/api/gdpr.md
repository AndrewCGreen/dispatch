# GDPR Endpoints

Dispatch provides dedicated endpoints for GDPR compliance — data subject access requests (export) and right to erasure (forget).

---

## Prerequisites

GDPR endpoints require:
- `compliance.gdpr_enabled: true` in `dispatch.yaml` (default)
- Master API key (these are cross-site operations)

---

## Export Data (Data Subject Access Request)

Returns all data Dispatch holds for an email address across all configured sites.

```
POST /api/v1/gdpr/export
```

### Request

```json
{
  "email": "user@example.com"
}
```

### Response (200 OK)

```json
{
  "email": "user@example.com",
  "sites": {
    "my-store": {
      "id": "sub_abc123",
      "site": "my-store",
      "email": "user@example.com",
      "name": "Alex Johnson",
      "status": "active",
      "attributes": { "plan": "pro" },
      "lists": ["newsletter", "product-updates"],
      "created_at": "2026-01-15T10:00:00Z",
      "confirmed_at": "2026-01-15T10:02:00Z"
    },
    "blog": {
      "id": "sub_def456",
      "site": "blog",
      "email": "user@example.com",
      "name": "Alex",
      "status": "unsubscribed",
      "attributes": {},
      "lists": ["weekly-digest"],
      "created_at": "2026-02-01T14:00:00Z",
      "unsubscribed_at": "2026-03-01T09:00:00Z"
    }
  }
}
```

If the email has no data on a site, that site is omitted from the response.

### What's Included

- Subscriber records (name, email, status, attributes) for each site
- List memberships
- Subscription timestamps (created, confirmed, unsubscribed)

### What's Not Included (But Logged)

- Consent records (retained for legal compliance)
- Message send history (available separately via the messages API)
- Suppression status (check via the suppression endpoint)

### Side Effects

- A consent record is created: `action: "export"`, with the requester's IP

---

## Forget (Right to Erasure)

Permanently deletes all data for an email address across all sites and prevents future sends.

```
POST /api/v1/gdpr/forget
```

### Request

```json
{
  "email": "user@example.com",
  "confirm": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | ✅ | Email address to erase |
| `confirm` | bool | ✅ | Must be `true` — safety guard against accidental erasure |

### Response (200 OK)

```json
{
  "email": "user@example.com",
  "status": "forgotten"
}
```

### What Happens

1. **Subscriber records deleted** — removed from all sites
2. **List memberships deleted** — cascade delete
3. **Suppression entry created** — `reason: "gdpr"` — permanently prevents re-adding this email
4. **Consent record created** — `action: "forget"` — audit trail that erasure was performed
5. **Queued messages blocked** — any pending messages to this email will be silently skipped by queue workers (suppression check)

### What's Preserved

- **Consent log entries** — these are retained because GDPR Article 7 requires you to be able to demonstrate that consent was obtained (and later withdrawn). The consent log is your evidence.
- **Anonymized message logs** — message records remain (for aggregate statistics) but the email address is already suppressed from future sends.

### Safety: The `confirm` Flag

The `confirm: true` flag is required to prevent accidental erasure. Without it:

```json
{
  "error": "CONFIRMATION_REQUIRED",
  "message": "Set 'confirm': true to proceed",
  "code": "CONFIRMATION_REQUIRED"
}
```

---

## Implementation Notes for Your Application

### Handling GDPR Requests

When a user requests data export or deletion through your application:

1. **Verify their identity** — confirm they own the email address (e.g., require login or email verification)
2. **Call the Dispatch API** — use the export or forget endpoint
3. **Delete from your own systems** — Dispatch only manages email data; your app likely has user data too
4. **Respond within 30 days** — GDPR requires a response within one month

### Example: Python Workflow

```python
from dispatch import Dispatch

client = Dispatch(host="https://mail.yourdomain.com", api_key="dsp_master_xxx")

# User requests their data
export = client.gdpr.export("user@example.com")
# → Return this data to the user

# User requests deletion
client.gdpr.forget("user@example.com")
# → Also delete from your own database
# → Confirm to the user that their data has been erased
```

### Suppression After Forget

After a GDPR forget, the email is permanently suppressed. If you try to send to them:

```bash
POST /api/v1/sites/my-site/send
{"to": "forgotten-user@example.com", "template": "welcome"}

# Response: 409
{
  "error": "RECIPIENT_SUPPRESSED",
  "message": "Recipient is on the suppression list",
  "code": "RECIPIENT_SUPPRESSED"
}
```

If you try to re-subscribe them:

```bash
POST /api/v1/sites/my-site/subscribers
{"email": "forgotten-user@example.com"}

# Response: 409
{
  "error": "RECIPIENT_SUPPRESSED",
  "message": "Email is on the suppression list",
  "code": "RECIPIENT_SUPPRESSED"
}
```

To re-add a GDPR-forgotten email, you must:
1. Obtain fresh, explicit consent from the person
2. Remove the suppression: `DELETE /api/v1/suppressions/forgotten-user@example.com`
3. Re-subscribe them with new consent information
