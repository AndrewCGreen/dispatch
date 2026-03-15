// Package backend defines the email delivery backend interface and registry.
//
// All backends implement the Backend interface. New backends are registered
// via Register() in their init() function, making them available when the
// package is imported. The Router maps site slugs to backend instances.
//
// Built-in backends:
//   - smtp: Direct SMTP delivery (smtp.go)
//   - resend: Resend API (resend.go)
//
// To add a new backend, implement Backend, call Register() in init(), and
// import the package in cmd/serve.go.
package backend

import (
	"context"
	"fmt"
	"sync"

	"github.com/dispatch-email/dispatch/internal/config"
)

// OutgoingMessage is a fully rendered email ready for delivery.
type OutgoingMessage struct {
	MessageID string
	From      string
	FromName  string
	ReplyTo   string
	To        string
	Subject   string
	HTML      string
	Text      string
	Headers   map[string]string
	Tags      []string
	Metadata  map[string]string
}

// SendResult is returned after a successful send.
type SendResult struct {
	BackendID string // provider's message ID
	Status    string // "sent", "queued"
}

// BatchResult is returned after a batch send.
type BatchResult struct {
	Succeeded int
	Failed    int
	Errors    []BatchError
}

// BatchError describes a single failure in a batch.
type BatchError struct {
	Index int
	Email string
	Error error
}

// Backend is the interface that all email delivery backends implement.
type Backend interface {
	// Name returns the backend identifier.
	Name() string

	// Send delivers a single rendered email.
	Send(ctx context.Context, msg *OutgoingMessage) (*SendResult, error)

	// BatchSend delivers multiple emails. Implementations can optimize
	// for their provider's batch API. Default falls back to sequential Send.
	BatchSend(ctx context.Context, msgs []*OutgoingMessage) (*BatchResult, error)

	// Health checks backend connectivity.
	Health(ctx context.Context) error

	// MaxBatchSize returns the maximum batch size (0 = no batch support).
	MaxBatchSize() int
}

// --- Registry ---

// Factory creates a Backend from config values.
type Factory func(cfg map[string]string) (Backend, error)

var (
	registry   = map[string]Factory{}
	registryMu sync.RWMutex
)

// Register adds a backend factory to the global registry.
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Create instantiates a backend by name with the given config.
func Create(name string, cfg map[string]string) (Backend, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown backend: %q (registered: %v)", name, registeredNames())
	}
	return factory(cfg)
}

func registeredNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var names []string
	for k := range registry {
		names = append(names, k)
	}
	return names
}

// --- Router ---

// Router maps site slugs to their configured backend.
type Router struct {
	backends map[string]Backend // backend name → instance
	siteMap  map[string]string  // site slug → backend name
}

// NewRouter creates a backend router from config.
func NewRouter() *Router {
	return &Router{
		backends: make(map[string]Backend),
		siteMap:  make(map[string]string),
	}
}

// AddBackend registers a backend instance.
func (r *Router) AddBackend(b Backend) {
	r.backends[b.Name()] = b
}

// MapSite associates a site with a backend.
func (r *Router) MapSite(siteSlug, backendName string) {
	r.siteMap[siteSlug] = backendName
}

// ForSite returns the backend configured for a given site.
func (r *Router) ForSite(siteSlug string) (Backend, error) {
	name, ok := r.siteMap[siteSlug]
	if !ok {
		return nil, fmt.Errorf("no backend configured for site %q", siteSlug)
	}
	b, ok := r.backends[name]
	if !ok {
		return nil, fmt.Errorf("backend %q not found (configured for site %q)", name, siteSlug)
	}
	return b, nil
}

// HealthAll checks all registered backends.
func (r *Router) HealthAll(ctx context.Context) map[string]string {
	results := make(map[string]string)
	for name, b := range r.backends {
		if err := b.Health(ctx); err != nil {
			results[name] = fmt.Sprintf("error: %v", err)
		} else {
			results[name] = "ok"
		}
	}
	return results
}

// InitFromConfig creates and wires up all backends from the config.
func InitFromConfig(cfg *config.Config) (*Router, error) {
	router := NewRouter()

	// Collect unique backends needed
	needed := map[string]map[string]string{}
	for _, site := range cfg.Sites {
		if _, exists := needed[site.Backend]; !exists {
			needed[site.Backend] = site.BackendCfg
		}
		router.MapSite(site.Slug, site.Backend)
	}

	// Create each backend
	for name, bcfg := range needed {
		b, err := Create(name, bcfg)
		if err != nil {
			return nil, fmt.Errorf("creating backend %q: %w", name, err)
		}
		router.AddBackend(b)
	}

	return router, nil
}
