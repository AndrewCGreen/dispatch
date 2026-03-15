# HTTP / cURL Examples

Reference examples for interacting with the Dispatch API using raw HTTP. These work with any HTTP client in any language.

---

## Base URL and Headers

```bash
# All requests need these headers:
BASE="https://mail.yourdomain.com"
KEY="dsp_site_xxxxxxxxxxxxx"

curl -H "Authorization: Bearer $KEY" \
     -H "Content-Type: application/json" \
     "$BASE/api/v1/..."
```

---

## Health Check (No Auth Required)

```bash
curl https://mail.yourdomain.com/api/v1/health
```

---

## Send Emails

### Template Send

```bash
curl -X POST "$BASE/api/v1/sites/my-site/send" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "template": "welcome",
    "data": {
      "name": "Alex",
      "login_url": "https://myapp.com/login"
    },
    "tags": ["onboarding"],
    "metadata": {"user_id": "usr_123"}
  }'
```

### Raw Send

```bash
curl -X POST "$BASE/api/v1/sites/my-site/send/raw" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "subject": "Your order shipped!",
    "html": "<h1>Shipped!</h1><p>Order: ORD-1234</p>",
    "text": "Shipped! Order: ORD-1234"
  }'
```

### Batch Send

```bash
curl -X POST "$BASE/api/v1/sites/my-site/send/batch" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "template": "weekly-digest",
    "recipients": [
      {"to": "alice@example.com", "data": {"name": "Alice"}},
      {"to": "bob@example.com",   "data": {"name": "Bob"}}
    ],
    "tags": ["digest"]
  }'
```

---

## Subscribers

### Add Subscriber

```bash
curl -X POST "$BASE/api/v1/sites/my-site/subscribers" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "name": "Alex Johnson",
    "lists": ["newsletter"],
    "consent": {
      "source": "signup-form",
      "ip": "203.0.113.42",
      "url": "https://mysite.com/signup"
    }
  }'
```

### Get Subscriber

```bash
curl "$BASE/api/v1/sites/my-site/subscribers/user@example.com" \
  -H "Authorization: Bearer $KEY"
```

### Delete Subscriber

```bash
curl -X DELETE "$BASE/api/v1/sites/my-site/subscribers/user@example.com" \
  -H "Authorization: Bearer $KEY"
```

---

## Suppressions

### Check Suppression

```bash
curl "$BASE/api/v1/suppressions/check/user@example.com" \
  -H "Authorization: Bearer $KEY"
```

### Add to Suppression

```bash
curl -X POST "$BASE/api/v1/suppressions" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "reason": "manual", "note": "Support request"}'
```

### Remove from Suppression

```bash
curl -X DELETE "$BASE/api/v1/suppressions/user@example.com" \
  -H "Authorization: Bearer $KEY"
```

---

## Templates

### List Templates

```bash
curl "$BASE/api/v1/sites/my-site/templates" \
  -H "Authorization: Bearer $KEY"
```

### Preview Template

```bash
curl -X POST "$BASE/api/v1/sites/my-site/templates/welcome/render" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"data": {"name": "Alex", "login_url": "https://example.com"}}'
```

---

## GDPR

### Export Data

```bash
curl -X POST "$BASE/api/v1/gdpr/export" \
  -H "Authorization: Bearer $MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

### Forget (Erase Data)

```bash
curl -X POST "$BASE/api/v1/gdpr/forget" \
  -H "Authorization: Bearer $MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "confirm": true}'
```

---

## Message Status

```bash
# Get full message details
curl "$BASE/api/v1/messages/msg_a1b2c3d4e5f6" \
  -H "Authorization: Bearer $KEY"

# Quick status check
curl "$BASE/api/v1/messages/msg_a1b2c3d4e5f6/status" \
  -H "Authorization: Bearer $KEY"
```

---

## Using with Other Languages

### Python (requests)

```python
import requests

BASE = "https://mail.yourdomain.com"
KEY = "dsp_site_xxx"
HEADERS = {"Authorization": f"Bearer {KEY}", "Content-Type": "application/json"}

resp = requests.post(
    f"{BASE}/api/v1/sites/my-site/send",
    headers=HEADERS,
    json={"to": "user@example.com", "template": "welcome", "data": {"name": "Alex"}}
)
resp.raise_for_status()
print(resp.json()["id"])
```

### JavaScript (fetch)

```javascript
const BASE = "https://mail.yourdomain.com";
const KEY = "dsp_site_xxx";

const resp = await fetch(`${BASE}/api/v1/sites/my-site/send`, {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${KEY}`,
    "Content-Type": "application/json"
  },
  body: JSON.stringify({
    to: "user@example.com",
    template: "welcome",
    data: { name: "Alex" }
  })
});

const result = await resp.json();
console.log(result.id);  // msg_a1b2c3d4e5f6
```

### PHP

```php
$ch = curl_init();
curl_setopt_array($ch, [
    CURLOPT_URL => "https://mail.yourdomain.com/api/v1/sites/my-site/send",
    CURLOPT_POST => true,
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_HTTPHEADER => [
        "Authorization: Bearer dsp_site_xxx",
        "Content-Type: application/json"
    ],
    CURLOPT_POSTFIELDS => json_encode([
        "to" => "user@example.com",
        "template" => "welcome",
        "data" => ["name" => "Alex"]
    ])
]);

$result = json_decode(curl_exec($ch), true);
echo $result["id"];
curl_close($ch);
```

### Ruby

```ruby
require 'net/http'
require 'json'
require 'uri'

uri = URI("https://mail.yourdomain.com/api/v1/sites/my-site/send")
req = Net::HTTP::Post.new(uri)
req["Authorization"] = "Bearer dsp_site_xxx"
req["Content-Type"] = "application/json"
req.body = {to: "user@example.com", template: "welcome", data: {name: "Alex"}}.to_json

resp = Net::HTTP.start(uri.hostname, uri.port, use_ssl: true) { |http| http.request(req) }
result = JSON.parse(resp.body)
puts result["id"]
```
