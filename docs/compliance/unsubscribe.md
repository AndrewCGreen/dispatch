# Unsubscribe Handling

Dispatch provides multiple unsubscribe mechanisms to comply with email regulations and respect recipient preferences.

---

## Unsubscribe Methods

### 1. Unsubscribe Link in Email Body

Include `{{.UnsubscribeURL}}` in your templates:

```html
<a href="{{.UnsubscribeURL}}">Unsubscribe from this list</a>
```

This generates a tokenized URL like:
```
https://mail.yourdomain.com/unsubscribe/eyJhbGciOiJIUzI1NiIs...
```

The token contains the subscriber ID and site — no login required.

### 2. List-Unsubscribe Header (RFC 8058)

Dispatch automatically adds the `List-Unsubscribe` and `List-Unsubscribe-Post` headers to all list/campaign emails:

```
List-Unsubscribe: <https://mail.yourdomain.com/unsubscribe/TOKEN>
List-Unsubscribe-Post: List-Unsubscribe=One-Click
```

This enables one-click unsubscribe in email clients that support it (Gmail, Apple Mail, Outlook, Yahoo). The recipient sees an "Unsubscribe" button directly in their mail client.

### 3. Manual API Call

Your application can unsubscribe a user directly:

```bash
DELETE /api/v1/sites/my-site/subscribers/user@example.com
```

---

## Unsubscribe Flow

```
Recipient clicks unsubscribe link
         │
         ▼
┌─────────────────────────────────────┐
│  GET /unsubscribe/{token}           │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  "You'll be unsubscribed    │   │
│  │   from [Site Name]."        │   │
│  │                             │   │
│  │  [Confirm Unsubscribe]      │   │
│  │                             │   │
│  │  Want to manage which       │   │
│  │  emails you receive?        │   │
│  │  [Manage Preferences]       │   │
│  └─────────────────────────────┘   │
└─────────────────┬───────────────────┘
                  │
         Clicks confirm
                  │
                  ▼
┌─────────────────────────────────────┐
│  POST /unsubscribe/{token}          │
│                                     │
│  1. Mark subscriber: unsubscribed   │
│  2. Add to suppression list         │
│  3. Log consent: unsubscribe        │
│  4. Fire webhook: subscriber.unsub  │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  "You've been unsubscribed. │   │
│  │   You won't receive any     │   │
│  │   more emails from us."     │   │
│  │                             │   │
│  │  Changed your mind?         │   │
│  │  [Resubscribe]              │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

---

## Customizing the Unsubscribe Page

The default unsubscribe page is functional but minimal. To customize it, you can (planned):

1. **Override the template** — place an `unsubscribe.html` in `shared/templates/`
2. **Redirect to your own page** — configure a custom unsubscribe URL that calls the Dispatch API

---

## Transactional Emails

Transactional emails (password resets, order confirmations) should **not** include unsubscribe links because:
- They're legally exempt from unsubscribe requirements
- Unsubscribing from password resets would be a security risk
- It confuses recipients

Mark transactional lists as non-unsubscribable:

```yaml
lists:
  - slug: transactional
    name: Transactional
    unsubscribable: false
```

Dispatch will not inject unsubscribe headers or links for messages sent to these lists.

---

## Resubscription

If a recipient unsubscribes and later wants to resubscribe:

1. They must provide fresh, explicit consent (GDPR requirement)
2. Remove them from the suppression list: `DELETE /api/v1/suppressions/{email}`
3. Create a new subscription: `POST /api/v1/sites/{site}/subscribers`

For GDPR-forgotten users, the process is the same — obtain consent first, then remove suppression and resubscribe.
