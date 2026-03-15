# Messages

Query sent messages and track delivery status.

---

## List Messages

```
GET /api/v1/sites/{site}/messages
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `per_page` | int | 50 | Items per page (max 200) |
| `status` | string | — | Filter by status: `queued`, `delivered`, `bounced`, `failed` |
| `template` | string | — | Filter by template slug |

### Response (200 OK)

```json
{
  "messages": [
    {
      "id": "msg_a1b2c3d4e5f6",
      "site": "my-site",
      "to": "user@example.com",
      "template": "welcome",
      "subject": "Welcome to My Site!",
      "status": "delivered",
      "backend": "resend",
      "backend_id": "re_xxxxx",
      "tags": ["onboarding"],
      "queued_at": "2026-03-14T22:50:00Z",
      "sent_at": "2026-03-14T22:50:01Z",
      "delivered_at": "2026-03-14T22:50:03Z"
    }
  ],
  "total": 1523
}
```

---

## Get Message

```
GET /api/v1/messages/{id}
```

### Response (200 OK)

```json
{
  "id": "msg_a1b2c3d4e5f6",
  "site": "my-site",
  "to": "user@example.com",
  "template": "welcome",
  "subject": "Welcome to My Site!",
  "status": "delivered",
  "backend": "resend",
  "backend_id": "re_xxxxx",
  "tags": ["onboarding"],
  "metadata": { "user_id": "usr_123" },
  "queued_at": "2026-03-14T22:50:00Z",
  "sent_at": "2026-03-14T22:50:01Z",
  "delivered_at": "2026-03-14T22:50:03Z"
}
```

For failed messages, the `error` field contains the last error:

```json
{
  "id": "msg_xyz789",
  "status": "failed",
  "error": "resend API error 422: domain not verified",
  "failed_at": "2026-03-14T23:50:00Z"
}
```

---

## Get Message Status (Quick)

A lightweight endpoint that returns only the status.

```
GET /api/v1/messages/{id}/status
```

### Response (200 OK)

```json
{
  "id": "msg_a1b2c3d4e5f6",
  "status": "delivered"
}
```

### Message Status Values

| Status | Description |
|--------|-------------|
| `queued` | Accepted and waiting in the send queue |
| `sending` | A worker has picked it up and is delivering |
| `delivered` | Backend confirmed delivery to the recipient's mail server |
| `bounced` | Recipient's mail server rejected the email |
| `failed` | All retry attempts exhausted — permanent failure |
| `suppressed` | Recipient was on the suppression list — not sent |
