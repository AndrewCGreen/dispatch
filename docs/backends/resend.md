# Resend Backend

[Resend](https://resend.com) is a transactional email API with a developer-friendly interface, excellent deliverability, and a generous free tier. It's the recommended backend for most Dispatch deployments.

---

## Why Resend?

- **Free tier:** 3,000 emails/month, 100/day — enough to start
- **Easy DNS setup:** Simple SPF/DKIM verification in their dashboard
- **Good deliverability:** Modern infrastructure with strong IP reputation
- **Clean API:** Simple, well-documented REST API
- **Fast setup:** Domain verified in minutes

---

## Setup

### 1. Create a Resend Account

Sign up at [resend.com](https://resend.com). It's free to start.

### 2. Verify Your Sending Domain

In the Resend dashboard:
1. Go to **Domains** → **Add Domain**
2. Enter your sending domain (e.g., `mysite.com`)
3. Add the provided DNS records (SPF, DKIM, DMARC)
4. Wait for verification (usually a few minutes)

### 3. Create an API Key

In the Resend dashboard:
1. Go to **API Keys** → **Create API Key**
2. Name it (e.g., "Dispatch Production")
3. Set permission to **Full Access** (or **Sending Access** if you prefer minimal permissions)
4. Copy the key

### 4. Configure Dispatch

```yaml
# sites/my-site/site.yaml
slug: my-site
from: hello@mysite.com          # Must be on your verified domain
from_name: "My Site"
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"  # Set this env var
```

Set the environment variable:
```bash
export RESEND_API_KEY=re_xxxxxxxxxxxxxxxxxx
```

---

## Configuration Reference

```yaml
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"  # Required: Resend API key (starts with "re_")
```

| Field | Required | Description |
|-------|----------|-------------|
| `api_key` | ✅ | Resend API key. Always use environment variable — never hardcode. |

---

## Resend Free Tier Limits

| Limit | Value |
|-------|-------|
| Monthly emails | 3,000 |
| Daily emails | 100 |
| Sending domains | 1 |
| API keys | 1 |

For higher volume, see [Resend Pricing](https://resend.com/pricing).

---

## Testing

Use Resend's built-in test functionality. When developing, send to a verified email address to confirm delivery without burning quota.

Resend also provides a **test API key** (starts with `re_test_`) that accepts API calls but doesn't actually send emails — useful for integration testing.

---

## Webhook Integration (Planned)

Resend fires webhooks for delivery events (delivered, bounced, complained). Dispatch will connect these to the webhook system in v0.2:

1. In Resend dashboard: **Webhooks** → **Add Endpoint**
2. URL: `https://mail.yourdomain.com/webhooks/resend`
3. Events: `email.delivered`, `email.bounced`, `email.complained`

This enables automatic bounce and complaint handling — bounced addresses will be added to the suppression list automatically.

---

## Health Check

The Resend backend health check calls the Resend API's `/domains` endpoint with your API key:

```json
{
  "backends": {
    "resend": "ok"
  }
}
```

If the key is invalid:
```json
{
  "backends": {
    "resend": "error: resend API returned 401"
  }
}
```

---

## Troubleshooting

### 401 Unauthorized
```
resend API error 401: Unauthorized
```
- Check your API key is correct
- Ensure the `RESEND_API_KEY` environment variable is set
- Verify the key hasn't been revoked in the Resend dashboard

### 422 Domain Not Verified
```
resend API error 422: The domain is not verified
```
- Your `from` address must use a domain verified in Resend
- Check the domain verification status in the Resend dashboard
- DNS propagation can take up to 48 hours after adding records

### 429 Rate Limited
```
resend API error 429: Too many requests
```
- You've hit the daily or monthly sending limit
- Upgrade your Resend plan or wait until the limit resets
- Dispatch will retry rate-limited sends with exponential backoff
