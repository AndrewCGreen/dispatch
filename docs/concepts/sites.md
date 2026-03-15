# Sites

Sites are the core organizational unit in Dispatch. Each site represents one of your web applications, products, or brands that sends email.

---

## Overview

A site maps to:
- A **sender identity** (from address, display name)
- An **email backend** (how emails are delivered)
- A **set of templates** (what the emails look like)
- A **set of subscribers** (who receives the emails)
- An **API key** (how your application authenticates)

Sites are fully isolated from each other. A subscriber on Site A is separate from the same email on Site B. Templates, API keys, and backend configs are all per-site.

---

## Creating a Site

### Via CLI

```bash
dispatch site add my-store
```

This creates:
```
sites/my-store/
├── site.yaml
└── templates/
    └── welcome.html
```

### Manually

Create the directory structure and `site.yaml` yourself:

```bash
mkdir -p sites/my-store/templates
```

Write `sites/my-store/site.yaml`:

```yaml
slug: my-store
name: My Online Store
from: orders@mystore.com
from_name: "My Store"
reply_to: support@mystore.com
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"
templates_dir: ./templates
double_optin: false
api_key: "dsp_site_mystore_xxxxxxxxxxxx"
```

---

## Multi-Site Patterns

### Pattern 1: Same Backend, Different Senders

All sites use Resend, but each sends from a different domain:

```yaml
# sites/store/site.yaml
slug: store
from: orders@mystore.com
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"

# sites/blog/site.yaml
slug: blog
from: newsletter@myblog.com
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"

# sites/saas/site.yaml
slug: saas
from: noreply@mysaas.io
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"
```

Dispatch deduplicates backends — all three sites share one Resend connection internally.

### Pattern 2: Mixed Backends

Different sites use different providers based on needs:

```yaml
# sites/main-product/site.yaml
# High volume — use SES for cost efficiency
backend: ses
backend_config:
  region: us-east-1

# sites/side-project/site.yaml
# Low volume — use SMTP through your existing mail server
backend: smtp
backend_config:
  host: mail.myserver.com
  port: "587"
  username: "${SMTP_USER}"
  password: "${SMTP_PASS}"
```

### Pattern 3: Development vs Production

Use environment variables to switch backends:

```yaml
# sites/my-site/site.yaml
backend: "${EMAIL_BACKEND}"
backend_config:
  api_key: "${EMAIL_API_KEY}"
  host: "${SMTP_HOST}"
```

In development:
```bash
EMAIL_BACKEND=smtp SMTP_HOST=localhost dispatch serve
```

In production:
```bash
EMAIL_BACKEND=resend EMAIL_API_KEY=re_xxxxx dispatch serve
```

---

## Managing Sites

### Listing Sites

```bash
dispatch site list
```

Output:
```
Sites:
  • my-store
  • blog
  • saas-app
```

### Removing a Site

```bash
dispatch site remove my-store
```

This removes the site's configuration and templates. **It does not delete subscriber data from the database.** To fully remove a site's data, use the GDPR forget endpoint or delete records directly.

---

## Site API Scoping

API keys are scoped to their site. A site API key can only access its own data:

```bash
# This works (matching site)
curl -H "Authorization: Bearer dsp_site_mystore_xxx" \
  http://localhost:8080/api/v1/sites/my-store/send

# This fails (wrong site)
curl -H "Authorization: Bearer dsp_site_mystore_xxx" \
  http://localhost:8080/api/v1/sites/blog/send
# → 403 Forbidden: No access to this site
```

The master key has access to all sites:

```bash
# Master key works for any site
curl -H "Authorization: Bearer dsp_master_xxx" \
  http://localhost:8080/api/v1/sites/blog/send
```

---

## Site Discovery

Dispatch discovers sites at startup by scanning the `sites/` directory. Each subdirectory containing a `site.yaml` file is loaded as a site.

Discovery order:
1. Read all directories under `sites/`
2. For each directory, attempt to load `site.yaml`
3. Parse the config, expand environment variables
4. Validate required fields (`slug`, `from`, `backend`)
5. Register the site in the config map

If a site config fails to load, Dispatch logs the error and **refuses to start**. This prevents silent misconfiguration.

---

## Best Practices

1. **One site per product/brand** — don't mix different products in one site
2. **Use environment variables for secrets** — never hardcode API keys in `site.yaml`
3. **Version control your site configs** — treat them like code
4. **Use descriptive slugs** — `my-store` not `site1`
5. **Set up DNS properly** — SPF, DKIM, and DMARC for each sending domain
6. **Test with a separate site** — create a `dev` or `staging` site for testing
