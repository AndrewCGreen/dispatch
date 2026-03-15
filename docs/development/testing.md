# Testing

Writing and running tests for Dispatch.

---

## Running Tests

```bash
# All tests
CGO_ENABLED=1 go test ./...

# Specific package
go test ./internal/store/...
go test ./internal/template/...
go test ./internal/backend/...

# With verbose output
go test -v ./...

# With race detector (always use for concurrent code)
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Test Structure

```
dispatch/
├── internal/
│   ├── config/
│   │   └── config_test.go
│   ├── store/
│   │   └── store_test.go
│   ├── template/
│   │   └── engine_test.go
│   ├── backend/
│   │   ├── smtp_test.go
│   │   └── resend_test.go
│   └── server/
│       └── server_test.go
└── tests/
    └── integration/
        └── send_test.go
```

---

## Unit Tests

### Testing the Store

```go
// internal/store/store_test.go

package store_test

import (
    "context"
    "testing"

    "github.com/dispatch-email/dispatch/internal/models"
    "github.com/dispatch-email/dispatch/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
    t.Helper()
    s, err := store.New("sqlite3", ":memory:")
    if err != nil {
        t.Fatal("failed to create test store:", err)
    }
    if err := s.Migrate(); err != nil {
        t.Fatal("failed to run migrations:", err)
    }
    t.Cleanup(func() { s.Close() })
    return s
}

func TestIsSuppressed(t *testing.T) {
    s := setupTestStore(t)
    ctx := context.Background()

    // Not suppressed initially
    suppressed, err := s.IsSuppressed(ctx, "user@example.com")
    if err != nil {
        t.Fatal(err)
    }
    if suppressed {
        t.Error("expected not suppressed")
    }

    // Add to suppression
    err = s.AddSuppression(ctx, &models.Suppression{
        Email:  "user@example.com",
        Reason: models.ReasonUnsubscribe,
    })
    if err != nil {
        t.Fatal(err)
    }

    // Now suppressed
    suppressed, err = s.IsSuppressed(ctx, "user@example.com")
    if err != nil {
        t.Fatal(err)
    }
    if !suppressed {
        t.Error("expected suppressed after adding")
    }
}

func TestCreateAndGetSubscriber(t *testing.T) {
    s := setupTestStore(t)
    ctx := context.Background()

    sub := &models.Subscriber{
        Site:   "test-site",
        Email:  "user@example.com",
        Name:   "Test User",
        Status: models.StatusActive,
        Lists:  []string{"newsletter"},
    }

    if err := s.CreateSubscriber(ctx, sub); err != nil {
        t.Fatal("failed to create subscriber:", err)
    }
    if sub.ID == "" {
        t.Error("expected ID to be set after create")
    }

    // Retrieve
    got, err := s.GetSubscriber(ctx, "test-site", "user@example.com")
    if err != nil {
        t.Fatal(err)
    }
    if got == nil {
        t.Fatal("expected subscriber, got nil")
    }
    if got.Name != "Test User" {
        t.Errorf("name: got %q, want %q", got.Name, "Test User")
    }
    if len(got.Lists) != 1 || got.Lists[0] != "newsletter" {
        t.Errorf("lists: got %v, want [newsletter]", got.Lists)
    }
}

func TestQueueLifecycle(t *testing.T) {
    s := setupTestStore(t)
    ctx := context.Background()

    // Create a message (also enqueues it)
    msg := &models.Message{
        Site:    "test-site",
        ToEmail: "user@example.com",
        Template: "welcome",
        Subject: "Welcome!",
        Backend: "smtp",
    }
    if err := s.CreateMessage(ctx, msg); err != nil {
        t.Fatal("failed to create message:", err)
    }

    // Dequeue
    items, err := s.DequeueMessages(ctx, "test-worker", 10)
    if err != nil {
        t.Fatal(err)
    }
    if len(items) != 1 {
        t.Fatalf("expected 1 queue item, got %d", len(items))
    }

    // Complete
    if err := s.CompleteQueueItem(ctx, items[0].ID, "backend-id-123"); err != nil {
        t.Fatal("failed to complete:", err)
    }

    // Should not appear again
    items, err = s.DequeueMessages(ctx, "test-worker", 10)
    if err != nil {
        t.Fatal(err)
    }
    if len(items) != 0 {
        t.Error("expected empty queue after completion")
    }

    // Check message status
    got, _ := s.GetMessage(ctx, msg.ID)
    if got.Status != models.MsgDelivered {
        t.Errorf("status: got %q, want %q", got.Status, models.MsgDelivered)
    }
    if got.BackendID != "backend-id-123" {
        t.Errorf("backend_id: got %q, want %q", got.BackendID, "backend-id-123")
    }
}
```

### Testing the Template Engine

```go
// internal/template/engine_test.go

package template_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/dispatch-email/dispatch/internal/template"
)

func setupTemplateDir(t *testing.T) (siteDir, sharedDir string) {
    t.Helper()
    dir := t.TempDir()
    siteDir = filepath.Join(dir, "templates")
    sharedDir = filepath.Join(dir, "shared")
    os.MkdirAll(siteDir, 0755)
    os.MkdirAll(sharedDir, 0755)
    return
}

func TestRenderSingleFile(t *testing.T) {
    siteDir, sharedDir := setupTemplateDir(t)

    // Write a test template
    tplContent := `{{/* subject: Hello, {{.Data.name}}! */}}
<h1>Welcome, {{.Data.name}}!</h1>`
    os.WriteFile(filepath.Join(siteDir, "welcome.html"), []byte(tplContent), 0644)

    engine := template.New(sharedDir)
    result, err := engine.Render(siteDir, "welcome", &template.TemplateData{
        Data: map[string]any{"name": "Alex"},
        Site: template.SiteInfo{Name: "Test Site"},
    })
    if err != nil {
        t.Fatal("render failed:", err)
    }

    if result.Subject != "Hello, Alex!" {
        t.Errorf("subject: got %q, want %q", result.Subject, "Hello, Alex!")
    }
    if result.HTML == "" {
        t.Error("expected non-empty HTML")
    }
}

func TestRenderMissingTemplate(t *testing.T) {
    _, sharedDir := setupTemplateDir(t)
    siteDir := t.TempDir()

    engine := template.New(sharedDir)
    _, err := engine.Render(siteDir, "nonexistent", &template.TemplateData{})
    if err == nil {
        t.Error("expected error for missing template")
    }
}
```

---

## Integration Tests

Integration tests run the full server against a real (in-memory SQLite) database:

```go
// tests/integration/send_test.go

package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/dispatch-email/dispatch/internal/config"
    "github.com/dispatch-email/dispatch/internal/server"
    "github.com/dispatch-email/dispatch/internal/store"
    // ...
)

func TestSendEndpoint(t *testing.T) {
    // Set up test server
    srv := setupTestServer(t)

    body := map[string]any{
        "to":       "user@example.com",
        "template": "welcome",
        "data":     map[string]any{"name": "Alex"},
    }
    bodyJSON, _ := json.Marshal(body)

    req := httptest.NewRequest("POST", "/api/v1/sites/test-site/send", bytes.NewReader(bodyJSON))
    req.Header.Set("Authorization", "Bearer dsp_site_test_key")
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)

    if w.Code != http.StatusAccepted {
        t.Errorf("status: got %d, want %d\nBody: %s", w.Code, http.StatusAccepted, w.Body)
    }

    var resp map[string]any
    json.NewDecoder(w.Body).Decode(&resp)

    if resp["status"] != "queued" {
        t.Errorf("expected status=queued, got %v", resp["status"])
    }
    if resp["id"] == "" {
        t.Error("expected non-empty message ID")
    }
}

func TestSendSuppressedRecipient(t *testing.T) {
    // ...test that suppressed recipients return 409
}

func TestSendMissingTemplate(t *testing.T) {
    // ...test that missing templates return 400
}
```

---

## Test Coverage Goals

| Package | Target Coverage |
|---------|----------------|
| `internal/store` | 90%+ |
| `internal/template` | 85%+ |
| `internal/backend` | 80%+ |
| `internal/server` | 75%+ |
| `internal/queue` | 75%+ |
| `internal/config` | 80%+ |
| Overall | 80%+ |

Check current coverage:
```bash
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```
