# Authentication

All API endpoints (except health and unsubscribe) require authentication via API key.

---

## API Key Types

| Type | Prefix | Scope | Use Case |
|------|--------|-------|----------|
| **Master** | `dsp_master_` | All sites, all endpoints | Admin operations, cross-site queries, key management |
| **Site** | `dsp_site_` | Single site only | Application integration — your app uses this |
| **Read-only** | `dsp_read_` | Read access only | Monitoring dashboards, analytics |

---

## Sending Authenticated Requests

Include the API key in the `Authorization` header:

```bash
curl -X POST http://localhost:8080/api/v1/sites/my-site/send \
  -H "Authorization: Bearer dsp_site_xxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"to": "user@example.com", "template": "welcome"}'
```

### Supported Auth Formats

```
Authorization: Bearer dsp_site_xxxxxxxxxxxxx
Authorization: Bearer dsp_master_xxxxxxxxxxxxx
```

---

## Key Sources

API keys can come from two places:

### 1. Config-Based Keys

Defined in configuration files — simplest for getting started:

**Master key** in `dispatch.yaml`:
```yaml
auth:
  master_key: "dsp_master_your_secure_key_here"
```

**Site key** in `sites/{site}/site.yaml`:
```yaml
api_key: "dsp_site_my_site_secure_key"
```

### 2. Database-Managed Keys (v0.2+)

Created and managed via API or CLI. Stored as bcrypt hashes in the database:

```bash
# Create a site key
dispatch keys create --site my-site --type site --name "Production API"

# Create a master key
dispatch keys create --type master --name "Admin Key"

# List all keys
dispatch keys list

# Revoke a key
dispatch keys revoke dsp_site_xxxxx
```

---

## Authorization Rules

### Site-Scoped Keys

A site key can only access endpoints for its own site:

```bash
# ✅ This works (key belongs to my-site)
GET /api/v1/sites/my-site/subscribers

# ❌ This fails with 403 (key belongs to my-site, not other-site)
GET /api/v1/sites/other-site/subscribers
```

### Master Keys

A master key has unrestricted access:

```bash
# ✅ All of these work with a master key
GET /api/v1/sites/my-site/subscribers
GET /api/v1/sites/other-site/subscribers
POST /api/v1/gdpr/forget
POST /api/v1/auth/keys
```

### Unauthenticated Endpoints

These endpoints work without authentication:

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/health` | Service health check (for load balancers) |
| `GET /unsubscribe/{token}` | Public unsubscribe page |
| `POST /unsubscribe/{token}` | Process unsubscribe |

---

## Error Responses

### Missing Authorization

```json
{
  "error": "UNAUTHORIZED",
  "message": "Missing Authorization header",
  "code": "UNAUTHORIZED"
}
```
HTTP Status: `401 Unauthorized`

### Invalid Key

```json
{
  "error": "INVALID_KEY",
  "message": "Invalid API key",
  "code": "INVALID_KEY"
}
```
HTTP Status: `401 Unauthorized`

### Insufficient Permissions

```json
{
  "error": "FORBIDDEN",
  "message": "No access to this site",
  "code": "FORBIDDEN"
}
```
HTTP Status: `403 Forbidden`

---

## Security Best Practices

1. **Use environment variables for keys** — never commit keys to version control
2. **Use site-scoped keys in applications** — principle of least privilege
3. **Reserve master key for admin operations** — don't use it in application code
4. **Rotate keys periodically** — create new key, update apps, revoke old key
5. **Use HTTPS in production** — API keys are transmitted in headers
6. **Monitor key usage** — `last_used` timestamp tracks when each key was last authenticated
