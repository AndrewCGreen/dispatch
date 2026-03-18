# Example: Send Raw HTML with Automatic Unsubscribe Footer

This example shows how to send emails with your own HTML while Dispatch handles the layout and unsubscribe footer automatically.

## Site Configuration

Add to your `sites/my-game/site.yaml`:

```yaml
slug: my-game
name: "My Awesome Game"
from: game@example.com
from_name: "My Game"

# Use any backend: smtp, resend, ses (future), etc.
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"

# Or use SMTP:
# backend: smtp
# backend_config:
#   host: smtp.gmail.com
#   port: 587
#   username: "${SMTP_USER}"
#   password: "${SMTP_PASS}"

# Configure the unsubscribe footer (works with ANY backend)
unsubscribe_footer:
  enabled: true
  # Customize the text (optional - has sensible defaults)
  text: "To stop receiving emails from {site_name}, click here: {unsubscribe_url}"
  # Available variables:
  # - {site_name}: Site display name
  # - {unsubscribe_url}: Unsubscribe URL
  # - {recipient_email}: Recipient's email address
```

**Note:** The unsubscribe footer feature works with **all backends** (SMTP, Resend, SES, etc.). The footer is added when the email is received via the API, before it reaches your configured backend.

## Sending Emails from Your Service

### Python Example

```python
import requests

DISPATCH_URL = "http://localhost:8080"
API_KEY = "your-site-api-key"

def send_reminder(email, player_name, game_stats):
    response = requests.post(
        f"{DISPATCH_URL}/api/v1/sites/my-game/send/raw",
        headers={
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        },
        json={
            "to": email,
            "subject": f"Daily Game Reminder - {player_name}!",
            "html": f"""
                <h1>Hello {player_name}!</h1>
                <p>Your daily game stats:</p>
                <ul>
                    <li>Score: {game_stats['score']}</li>
                    <li>Level: {game_stats['level']}</li>
                </ul>
                <p>Come back and play today!</p>
            """,
            "data": {
                "player_name": player_name,
                "stats": game_stats
            }
        }
    )
    return response.json()

# Send to a player
result = send_reminder(
    email="player@example.com",
    player_name="Alex",
    game_stats={"score": 1500, "level": 5}
)
print(f"Email queued: {result['id']}")
```

### JavaScript/Node.js Example

```javascript
async function sendReminder(email, playerName, gameStats) {
    const response = await fetch('http://localhost:8080/api/v1/sites/my-game/send/raw', {
        method: 'POST',
        headers: {
            'Authorization': 'Bearer your-site-api-key',
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            to: email,
            subject: `Daily Game Reminder - ${playerName}!`,
            html: `
                <h1>Hello ${playerName}!</h1>
                <p>Your daily game stats:</p>
                <ul>
                    <li>Score: ${gameStats.score}</li>
                    <li>Level: ${gameStats.level}</li>
                </ul>
                <p>Come back and play today!</p>
            `,
            data: {
                player_name: playerName,
                stats: gameStats
            }
        })
    });
    
    return await response.json();
}
```

### cURL Example

```bash
curl -X POST http://localhost:8080/api/v1/sites/my-game/send/raw \
  -H "Authorization: Bearer your-site-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "player@example.com",
    "subject": "Daily Game Reminder!",
    "html": "<h1>Hello Alex!</h1><p>Time to play your daily game!</p>",
    "data": {"player_name": "Alex"}
  }'
```

## What Dispatch Does

When you send via `/send/raw`:

1. **Renders your HTML** with any template variables from `data`
2. **Wraps it in a responsive email layout** (mobile-friendly)
3. **Adds the unsubscribe footer** at the bottom
4. **Generates plain text version** automatically (with footer)
5. **Adds List-Unsubscribe headers** for email client support
6. **Queues the email** for delivery

## Result

Your HTML:
```html
<h1>Hello Alex!</h1>
<p>Time to play!</p>
```

Becomes a fully formatted email with:
- ✅ Responsive layout (works on mobile)
- ✅ Professional email structure
- ✅ Unsubscribe link at bottom
- ✅ Plain text version
- ✅ Proper email headers

## Customizing Per Site

Each site can have its own footer configuration:

**Game site** (`sites/my-game/site.yaml`):
```yaml
unsubscribe_footer:
  text: "Don't want daily game reminders? Unsubscribe: {unsubscribe_url}"
```

**Fantasy sports site** (`sites/fantasy-sports/site.yaml`):
```yaml
unsubscribe_footer:
  text: "Manage your fantasy sports email preferences: {unsubscribe_url}"
```

This gives you the flexibility to keep all template control in your services while Dispatch handles the email infrastructure and compliance!
