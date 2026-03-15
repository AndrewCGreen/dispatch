# Templates API

List, view, and preview email templates. Templates are managed as files on disk — the API provides read-only access.

---

## List Templates

```
GET /api/v1/sites/{site}/templates
```

### Response (200 OK)

```json
{
  "templates": [
    { "slug": "welcome", "type": "single" },
    { "slug": "password-reset", "type": "single" },
    { "slug": "weekly-digest", "type": "directory" }
  ]
}
```

| Field | Description |
|-------|-------------|
| `slug` | Template identifier (used in send requests) |
| `type` | `single` (one .html file) or `directory` (body.html + subject.txt + body.txt) |

> Base templates (prefixed with `_`) are excluded from listings.

---

## Get Template Source

```
GET /api/v1/sites/{site}/templates/{slug}
```

### Response (200 OK)

```json
{
  "slug": "welcome",
  "source": "{{/* subject: Welcome to {{.Site.Name}}! */}}\n<html>...</html>"
}
```

---

## Preview Template

Render a template with sample data without sending an email.

```
POST /api/v1/sites/{site}/templates/{slug}/render
```

### Request

```json
{
  "data": {
    "name": "Alex",
    "login_url": "https://example.com/login"
  }
}
```

### Response (200 OK)

```json
{
  "subject": "Welcome to My Site!",
  "html": "<!DOCTYPE html><html>...<h1>Welcome, Alex!</h1>...</html>",
  "text": "Welcome, Alex!\n\nThanks for joining My Site..."
}
```

This is useful for:
- Testing template changes before deploying
- Generating previews in admin interfaces
- Validating template data requirements
