# Subscribers

Manage email subscribers per site — add, list, update, and remove subscribers with consent tracking.

---

## Add Subscriber

```
POST /api/v1/sites/{site}/subscribers
```

### Request

```json
{
  "email": "user@example.com",
  "name": "Alex Johnson",
  "attributes": {
    "plan": "pro",
    "signed_up": "2026-03-14",
    "referral_source": "twitter"
  },
  "lists": ["newsletter", "product-updates"],
  "consent": {
    "source": "signup-form",
    "ip": "203.0.113.42",
    "url": "https://mysite.com/signup"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | ✅ | Subscriber's email address |
| `name` | string | No | Subscriber's display name |
| `attributes` | object | No | Custom key-value attributes (stored as JSON) |
| `lists` | array | No | List slugs to subscribe to (must be defined in site config) |
| `consent` | object | No | Consent information for GDPR compliance |
| `consent.source` | string | No | How consent was obtained (e.g., "signup-form", "import", "api") |
| `consent.ip` | string | No | IP address of the subscriber at time of consent |
| `consent.url` | string | No | URL where the subscriber consented |

### Response — Without Double Opt-in (201 Created)

```json
{
  "email": "user@example.com",
  "status": "active"
}
```

### Response — With Double Opt-in (201 Created)

```json
{
  "email": "user@example.com",
  "status": "pending",
  "confirm_sent": true,
  "message": "Double opt-in confirmation sent"
}
```

### Error Responses

**Already exists (409):**
```json
{
  "error": "ALREADY_EXISTS",
  "message": "Subscriber already exists",
  "code": "ALREADY_EXISTS"
}
```

**Email suppressed (409):**
```json
{
  "error": "RECIPIENT_SUPPRESSED",
  "message": "Email is on the suppression list",
  "code": "RECIPIENT_SUPPRESSED"
}
```

---

## List Subscribers

```
GET /api/v1/sites/{site}/subscribers
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `per_page` | int | 50 | Items per page (max 200) |
| `status` | string | — | Filter by status: `pending`, `active`, `unsubscribed` |
| `list` | string | — | Filter by list slug |
| `search` | string | — | Search by email or name |

### Response (200 OK)

```json
{
  "subscribers": [
    {
      "id": "sub_abc123",
      "site": "my-site",
      "email": "user@example.com",
      "name": "Alex Johnson",
      "status": "active",
      "attributes": { "plan": "pro" },
      "lists": ["newsletter", "product-updates"],
      "created_at": "2026-03-14T10:00:00Z",
      "confirmed_at": "2026-03-14T10:02:00Z"
    }
  ],
  "total": 1432,
  "page": 1,
  "per_page": 50,
  "has_more": true
}
```

---

## Get Subscriber

```
GET /api/v1/sites/{site}/subscribers/{email}
```

### Response (200 OK)

```json
{
  "id": "sub_abc123",
  "site": "my-site",
  "email": "user@example.com",
  "name": "Alex Johnson",
  "status": "active",
  "attributes": { "plan": "pro", "signed_up": "2026-03-14" },
  "lists": ["newsletter", "product-updates"],
  "created_at": "2026-03-14T10:00:00Z",
  "confirmed_at": "2026-03-14T10:02:00Z"
}
```

---

## Update Subscriber

```
PUT /api/v1/sites/{site}/subscribers/{email}
```

### Request

```json
{
  "name": "Alex J.",
  "attributes": {
    "plan": "enterprise"
  },
  "lists": ["newsletter", "product-updates", "enterprise-features"]
}
```

Only provided fields are updated. Omitted fields remain unchanged.

### Response (200 OK)

```json
{
  "status": "updated"
}
```

---

## Delete Subscriber

Removes the subscriber and logs the unsubscribe action.

```
DELETE /api/v1/sites/{site}/subscribers/{email}
```

### Response (200 OK)

```json
{
  "status": "deleted"
}
```

This endpoint:
1. Deletes the subscriber record and list memberships
2. Logs a consent record (action: `unsubscribe`) if consent logging is enabled
3. **Does not** add to the suppression list (use the suppression endpoint for that)

---

## Subscriber Lifecycle

```
                   POST /subscribers
                         │
           ┌─────────────┼─────────────┐
           │                            │
     double_optin: true          double_optin: false
           │                            │
      ┌────▼─────┐                ┌─────▼────┐
      │ PENDING  │                │  ACTIVE  │
      └────┬─────┘                └──────────┘
           │
    clicks confirm link
           │
      ┌────▼─────┐
      │  ACTIVE  │
      └────┬─────┘
           │
    unsubscribes or
    DELETE /subscribers/{email}
           │
      ┌────▼─────────┐
      │ UNSUBSCRIBED │
      └────┬─────────┘
           │
    POST /gdpr/forget
           │
      ┌────▼─────────┐
      │   DELETED +  │
      │  SUPPRESSED  │
      └──────────────┘
```

---

## Consent Tracking

When consent logging is enabled (`compliance.consent_logging: true`), every subscriber action creates an immutable audit record:

```sql
-- Example consent log entry
INSERT INTO consent_log (id, email, site, action, source, ip, url, created_at)
VALUES ('uuid', 'user@example.com', 'my-site', 'subscribe', 'signup-form', '203.0.113.42', 'https://mysite.com/signup', '2026-03-14T10:00:00Z');
```

Consent records are **append-only** — they are never modified or deleted (even during GDPR forget, the consent log is preserved for legal compliance).
