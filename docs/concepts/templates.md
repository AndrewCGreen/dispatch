# Templates

Dispatch uses Go's `html/template` package for email rendering. Templates are files on disk — version-controlled, git-friendly, and editable with any text editor.

---

## Template Formats

### Single-File Template

The simplest format — one `.html` file per template:

```
sites/my-site/templates/
├── welcome.html
├── password-reset.html
└── order-confirmation.html
```

**Example: `welcome.html`**

```html
{{/* subject: Welcome to {{.Site.Name}}, {{.Data.name}}! */}}

<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body>
  <h1>Welcome, {{.Data.name}}!</h1>
  <p>Thanks for joining {{.Site.Name}}.</p>

  {{if .Data.login_url}}
  <p><a href="{{.Data.login_url}}">Get Started →</a></p>
  {{end}}

  <p style="color:#888;font-size:12px;">
    {{if .UnsubscribeURL}}<a href="{{.UnsubscribeURL}}">Unsubscribe</a>{{end}}
  </p>
</body>
</html>
```

The subject line is extracted from the first Go template comment matching `{{/* subject: ... */}}`.

### Directory-Based Template

For more complex templates with separate subject, HTML, and plain text versions:

```
sites/my-site/templates/
└── weekly-digest/
    ├── subject.txt       # Subject line template
    ├── body.html         # HTML version
    └── body.txt          # Plain text version (optional)
```

**`subject.txt`:**
```
Your weekly digest — {{.Data.items_count}} new items
```

**`body.html`:**
```html
<h1>Weekly Digest</h1>
<p>Hey {{.Data.name}}, here's what happened this week:</p>
<ul>
  {{range .Data.items}}
  <li><a href="{{.url}}">{{.title}}</a></li>
  {{end}}
</ul>
```

**`body.txt`:**
```
Weekly Digest

Hey {{.Data.name}}, here's what happened this week:

{{range .Data.items}}
- {{.title}}: {{.url}}
{{end}}
```

If `body.txt` is not provided, Dispatch auto-generates a plain text version by stripping HTML tags.

---

## Template Variables

Every template receives a `TemplateData` struct with these fields:

### `.Data` — Custom Data

Arbitrary key-value data passed in the send request:

```json
{
  "to": "user@example.com",
  "template": "welcome",
  "data": {
    "name": "Alex",
    "login_url": "https://example.com/login",
    "plan": "pro"
  }
}
```

Access in templates:
```html
<p>Hello {{.Data.name}}, you're on the {{.Data.plan}} plan.</p>
<a href="{{.Data.login_url}}">Log in</a>
```

### `.Site` — Site Information

| Variable | Type | Description |
|----------|------|-------------|
| `.Site.Name` | string | Site display name from `site.yaml` |
| `.Site.URL` | string | Site base URL |

```html
<p>Thanks for using {{.Site.Name}}!</p>
```

### `.Subscriber` — Recipient Information

| Variable | Type | Description |
|----------|------|-------------|
| `.Subscriber.Email` | string | Recipient's email address |
| `.Subscriber.Name` | string | Recipient's name (if known) |
| `.Subscriber.Attributes` | map | Custom subscriber attributes |

```html
<p>This email was sent to {{.Subscriber.Email}}</p>
```

### `.UnsubscribeURL` — Unsubscribe Link

A tokenized URL that allows one-click unsubscribe:

```html
<a href="{{.UnsubscribeURL}}">Unsubscribe from this list</a>
```

### `.PreferencesURL` — Email Preferences

A URL to the email preferences page (when configured):

```html
<a href="{{.PreferencesURL}}">Manage email preferences</a>
```

---

## Base Templates (Layouts)

Base templates provide shared layouts that other templates inherit from.

### Shared Base Template

Located at `shared/templates/_default-base.html`, available to all sites:

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { font-family: sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .content { background: #fff; border-radius: 8px; padding: 32px; }
    .footer { text-align: center; padding: 20px; color: #888; font-size: 12px; }
    .button { display: inline-block; padding: 12px 24px; background: #2563eb; color: #fff; text-decoration: none; border-radius: 6px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="content">
      {{block "content" .}}{{end}}
    </div>
    <div class="footer">
      <p>{{.Site.Name}}</p>
      {{if .UnsubscribeURL}}<a href="{{.UnsubscribeURL}}">Unsubscribe</a>{{end}}
    </div>
  </div>
</body>
</html>
```

### Site-Specific Base Template

Override the shared base for a specific site. Place in `sites/my-site/templates/_base.html`:

```html
<!-- Custom layout for this site with different branding -->
<!DOCTYPE html>
<html>
<head>
  <style>
    body { font-family: 'Georgia', serif; background: #1a1a2e; color: #eee; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    /* ... site-specific styles ... */
  </style>
</head>
<body>
  <div class="container">
    <img src="https://mysite.com/logo.png" alt="{{.Site.Name}}" width="150">
    {{block "content" .}}{{end}}
    <p style="font-size:11px;color:#666;">© 2026 {{.Site.Name}}</p>
  </div>
</body>
</html>
```

### Using a Base Template

Reference the base template in your email template:

```html
{{/* subject: Welcome, {{.Data.name}}! */}}

{{template "_base.html" .}}

{{define "content"}}
<h1>Welcome!</h1>
<p>Great to have you, {{.Data.name}}.</p>
{{end}}
```

Template resolution order:
1. Site-specific base templates (`sites/{site}/templates/_*.html`)
2. Shared base templates (`shared/templates/_*.html`)

---

## Go Template Syntax Reference

Dispatch uses Go's standard `html/template` package. Here's a quick reference:

### Variables

```html
{{.Data.name}}              <!-- Access data field -->
{{.Site.Name}}              <!-- Access site name -->
```

### Conditionals

```html
{{if .Data.is_premium}}
  <p>Premium member benefits...</p>
{{else}}
  <p>Upgrade to premium!</p>
{{end}}

{{if and .Data.name .Data.email}}
  <p>Hi {{.Data.name}} ({{.Data.email}})</p>
{{end}}
```

### Loops

```html
{{range .Data.items}}
  <li>{{.title}} — {{.price}}</li>
{{end}}

<!-- With index -->
{{range $i, $item := .Data.items}}
  <li>{{$i}}. {{$item.title}}</li>
{{end}}

<!-- Empty fallback -->
{{range .Data.items}}
  <li>{{.title}}</li>
{{else}}
  <li>No items found.</li>
{{end}}
```

### String Functions

```html
{{len .Data.items}}         <!-- Length of a slice/string -->
{{printf "%.2f" .Data.price}} <!-- Formatted output -->
```

### Nested Data

When your request data contains nested objects:

```json
{
  "data": {
    "order": {
      "id": "ORD-123",
      "items": [
        {"name": "Widget", "qty": 2, "price": 9.99},
        {"name": "Gadget", "qty": 1, "price": 24.99}
      ],
      "total": 44.97
    }
  }
}
```

Access in template:
```html
<h2>Order {{.Data.order.id}}</h2>
<table>
  {{range .Data.order.items}}
  <tr>
    <td>{{.name}}</td>
    <td>x{{.qty}}</td>
    <td>${{printf "%.2f" .price}}</td>
  </tr>
  {{end}}
</table>
<p><strong>Total: ${{printf "%.2f" .Data.order.total}}</strong></p>
```

---

## Template Preview

Test your templates without sending an email:

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/sites/my-site/templates/welcome/render \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "name": "Alex",
      "login_url": "https://example.com/login"
    }
  }'
```

Response:

```json
{
  "subject": "Welcome to My Site, Alex!",
  "html": "<html>...<h1>Welcome, Alex!</h1>...</html>",
  "text": "Welcome, Alex!\n\nThanks for joining..."
}
```

### Via CLI (planned)

```bash
dispatch template preview my-site welcome --data '{"name": "Alex"}'
```

---

## Email Design Best Practices

### Responsive Design

Use inline CSS (many email clients strip `<style>` tags):

```html
<table role="presentation" width="100%" style="max-width:600px;margin:0 auto;">
  <tr>
    <td style="padding:20px;font-family:sans-serif;">
      <!-- Content here -->
    </td>
  </tr>
</table>
```

### Dark Mode Support

```html
<style>
  @media (prefers-color-scheme: dark) {
    .content { background: #1a1a1a !important; color: #eee !important; }
    .button { background: #3b82f6 !important; }
  }
</style>
```

### Image Hosting

Don't embed images as base64 in emails — host them on your domain and reference with absolute URLs:

```html
<img src="https://mysite.com/email-assets/logo.png" alt="My Site" width="150">
```

### Preheader Text

Add hidden preheader text that shows in email client previews:

```html
<div style="display:none;max-height:0;overflow:hidden;">
  {{.Data.preheader_text}}
</div>
```

---

## Troubleshooting

### Template Not Found

```
TEMPLATE_ERROR: template "welcome" not found in sites/my-site/templates
```

Check:
- File exists at `sites/my-site/templates/welcome.html` or `sites/my-site/templates/welcome/body.html`
- File extension is `.html`
- `templates_dir` in `site.yaml` points to the correct directory

### Template Parse Error

```
TEMPLATE_ERROR: parsing template: unexpected "}" in command
```

Check your Go template syntax. Common issues:
- Missing closing `{{end}}`
- Unclosed `{{if}}` or `{{range}}` blocks
- Typos in variable names (Go templates fail silently on missing fields)

### Variables Not Rendering

If `{{.Data.name}}` renders as empty string:
- Check that you're passing the data in the API request
- Field names are case-sensitive
- Nested access uses dots: `{{.Data.order.id}}`
