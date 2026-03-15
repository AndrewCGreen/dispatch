# Suppressions

The global suppression list prevents emails from being sent to addresses that have unsubscribed, bounced, complained, or been GDPR-forgotten.

---

## Overview

The suppression list is the **last line of defense** against sending unwanted email. Before every send (both at API request time and in the queue worker), the recipient is checked against this list.

A suppressed email address is blocked across **all sites** when `compliance.suppression_global` is `true` (the default).

---

## List Suppressions

```
GET /api/v1/suppressions
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `per_page` | int | 50 | Items per page (max 200) |

### Response (200 OK)

```json
{
  "data": [
    {
      "email": "bounced@example.com",
      "reason": "bounce",
      "note": "550 5.1.1 User unknown",
      "created_at": "2026-03-10T15:00:00Z"
    },
    {
      "email": "unsubbed@example.com",
      "reason": "unsubscribe",
      "note": "",
      "created_at": "2026-03-12T09:30:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "per_page": 50,
  "has_more": false
}
```

---

## Add Suppression

Manually add an email to the suppression list.

```
POST /api/v1/suppressions
```

### Request

```json
{
  "email": "user@example.com",
  "reason": "manual",
  "note": "Requested removal via support ticket #4521"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | ✅ | Email address to suppress |
| `reason` | string | No | Reason for suppression (default: `manual`) |
| `note` | string | No | Human-readable note explaining the suppression |

### Suppression Reasons

| Reason | Description | Typical Source |
|--------|-------------|----------------|
| `unsubscribe` | Recipient clicked unsubscribe | Unsubscribe endpoint |
| `bounce` | Email address doesn't exist or is unreachable | Backend webhook |
| `complaint` | Recipient marked as spam | Backend webhook |
| `manual` | Manually added by an operator | API call |
| `gdpr` | Right to erasure request | GDPR forget endpoint |

### Response (201 Created)

```json
{
  "email": "user@example.com",
  "status": "suppressed"
}
```

---

## Remove Suppression

Remove an email from the suppression list, allowing future sends.

```
DELETE /api/v1/suppressions/{email}
```

### Response (200 OK)

```json
{
  "email": "user@example.com",
  "status": "removed"
}
```

> ⚠️ **Caution:** Removing a suppression for a `gdpr` reason may violate GDPR. Only remove GDPR suppressions if the person has explicitly re-consented.

---

## Check Suppression

Quick check whether a specific email is suppressed.

```
GET /api/v1/suppressions/check/{email}
```

### Response (200 OK)

```json
{
  "email": "user@example.com",
  "suppressed": true
}
```

Or:

```json
{
  "email": "other@example.com",
  "suppressed": false
}
```

---

## How Suppressions Are Created

Suppressions are created automatically by these events:

| Event | Reason | Description |
|-------|--------|-------------|
| Unsubscribe link clicked | `unsubscribe` | Recipient clicks the unsubscribe link in an email |
| Hard bounce | `bounce` | Backend reports the email address is invalid |
| Spam complaint | `complaint` | Recipient marks the email as spam |
| GDPR forget | `gdpr` | Data erasure request processed |
| Manual API call | `manual` | Operator adds via API or CLI |

---

## Suppression Enforcement

Suppressions are checked at two points:

### 1. At API Request Time

When a send request arrives, the recipient is checked immediately:

```
POST /api/v1/sites/my-site/send → Check suppression → 409 if suppressed
```

This gives your application immediate feedback that the recipient is blocked.

### 2. At Queue Processing Time

Even if the API check passes, the queue worker checks again before sending:

```
Queue worker → Load message → Check suppression → Skip if suppressed
```

This catches race conditions where a recipient is suppressed between the API request and queue processing.

---

## Global vs Per-Site Suppression

**Default (global):** When `compliance.suppression_global: true`, a suppression applies to all sites. If `user@example.com` unsubscribes from Site A, they won't receive email from Site B either.

**Per-site (planned):** When `compliance.suppression_global: false`, suppressions only apply to the site where they were created. This allows a user to unsubscribe from one product without affecting another.
