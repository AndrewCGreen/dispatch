# Custom Backends

Dispatch's backend system is fully pluggable. You can add any email provider by implementing the `Backend` interface.

---

## The Backend Interface

```go
// internal/backend/backend.go

type Backend interface {
    Name() string
    Send(ctx context.Context, msg *OutgoingMessage) (*SendResult, error)
    BatchSend(ctx context.Context, msgs []*OutgoingMessage) (*BatchResult, error)
    Health(ctx context.Context) error
    MaxBatchSize() int
}
```

---

## Step-by-Step: Adding a New Backend

Let's build a backend for a hypothetical provider called "MailBlast".

### Step 1: Create the Backend File

Create `internal/backend/mailblast.go`:

```go
package backend

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// Register the backend when the package is imported.
func init() {
    Register("mailblast", NewMailBlast)
}

// MailBlast implements the Backend interface.
type MailBlast struct {
    apiKey string
    client *http.Client
}

// NewMailBlast is the factory function called when a site uses backend: mailblast
func NewMailBlast(cfg map[string]string) (Backend, error) {
    apiKey := cfg["api_key"]
    if apiKey == "" {
        return nil, fmt.Errorf("mailblast backend requires api_key")
    }

    return &MailBlast{
        apiKey: apiKey,
        client: &http.Client{Timeout: 30 * time.Second},
    }, nil
}

// Name returns the backend identifier — must match site.yaml backend: field.
func (m *MailBlast) Name() string {
    return "mailblast"
}

// Send delivers a single email.
func (m *MailBlast) Send(ctx context.Context, msg *OutgoingMessage) (*SendResult, error) {
    // Build the payload for your provider's API
    payload := map[string]any{
        "from":    fmt.Sprintf("%s <%s>", msg.FromName, msg.From),
        "to":      msg.To,
        "subject": msg.Subject,
        "html":    msg.HTML,
        "text":    msg.Text,
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("marshaling payload: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mailblast.io/v1/send", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("X-API-Key", m.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := m.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("API request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("MailBlast API error: HTTP %d", resp.StatusCode)
    }

    // Parse provider's response to get their message ID
    var result struct {
        ID string `json:"id"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return &SendResult{
        BackendID: result.ID,
        Status:    "sent",
    }, nil
}

// BatchSend delivers multiple emails.
// If your provider has a batch API, implement it here.
// Otherwise, fall back to sequential sends.
func (m *MailBlast) BatchSend(ctx context.Context, msgs []*OutgoingMessage) (*BatchResult, error) {
    result := &BatchResult{}
    for i, msg := range msgs {
        if _, err := m.Send(ctx, msg); err != nil {
            result.Failed++
            result.Errors = append(result.Errors, BatchError{
                Index: i,
                Email: msg.To,
                Error: err,
            })
        } else {
            result.Succeeded++
        }
    }
    return result, nil
}

// Health checks that the backend is reachable and the API key is valid.
func (m *MailBlast) Health(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.mailblast.io/v1/account", nil)
    if err != nil {
        return err
    }
    req.Header.Set("X-API-Key", m.apiKey)

    resp, err := m.client.Do(req)
    if err != nil {
        return fmt.Errorf("MailBlast health check failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("MailBlast API returned %d", resp.StatusCode)
    }
    return nil
}

// MaxBatchSize returns the max batch size for this provider.
// Return 0 if you don't have a native batch API.
func (m *MailBlast) MaxBatchSize() int {
    return 0 // Falls back to sequential sends
}
```

### Step 2: Import the Backend

The `init()` function in your backend file calls `Register()` automatically, but only if the package is imported. Add the import to `cmd/serve.go`:

```go
import (
    // ... other imports ...

    // Import backends to trigger their init() registration
    _ "github.com/dispatch-email/dispatch/internal/backend/smtp"
    _ "github.com/dispatch-email/dispatch/internal/backend/resend"
    _ "github.com/dispatch-email/dispatch/internal/backend/mailblast" // Add this
)
```

> **Note:** In the current architecture, all backends are in the same `backend` package, so they're registered automatically. If you move backends to sub-packages in the future, explicit imports will be needed.

### Step 3: Configure a Site to Use It

```yaml
# sites/my-site/site.yaml
backend: mailblast
backend_config:
  api_key: "${MAILBLAST_API_KEY}"
```

### Step 4: Test It

```bash
# Check health
curl http://localhost:8080/api/v1/health

# Send a test email
dispatch send my-site welcome test@example.com
```

---

## The `OutgoingMessage` Struct

Your `Send()` method receives a fully rendered `OutgoingMessage`:

```go
type OutgoingMessage struct {
    MessageID  string            // Dispatch's internal ID (e.g., "msg_abc123")
    From       string            // Sender email (e.g., "hello@mysite.com")
    FromName   string            // Display name (e.g., "My Site")
    ReplyTo    string            // Reply-to address (may be empty)
    To         string            // Recipient email
    Subject    string            // Rendered subject line
    HTML       string            // Rendered HTML body
    Text       string            // Rendered plain text body
    Headers    map[string]string // Custom headers (List-Unsubscribe, etc.)
    Tags       []string          // Tags from the send request
    Metadata   map[string]string // Metadata from the send request
}
```

Standard headers automatically included:
- `List-Unsubscribe` — RFC 8058 unsubscribe
- `List-Unsubscribe-Post` — one-click unsubscribe
- `X-Dispatch-Message-ID` — Dispatch's message ID for tracing

---

## Publishing Your Backend

If your backend could be useful to others:

1. **Fork Dispatch** and add your backend to `internal/backend/`
2. **Submit a pull request** — we welcome new backends
3. **Publish as a Go module** — a separate module that imports the Dispatch backend interface

Community backends are listed in the [Dispatch ecosystem page](https://github.com/dispatch-email/dispatch/wiki/Community-Backends).
