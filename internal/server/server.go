// Package server implements the Dispatch HTTP API using the chi router.
//
// Middleware stack (applied in order):
//   1. RequestID — adds X-Request-ID to every request
//   2. RealIP — reads X-Real-IP / X-Forwarded-For for accurate client IPs
//   3. Recoverer — catches panics and returns 500
//   4. Timeout — 30s request timeout
//   5. logMiddleware — structured request logging
//   6. authMiddleware — API key validation (applied to /api/v1/* routes)
//
// Unauthenticated endpoints: GET /api/v1/health, GET/POST /unsubscribe/{token}
//
// All responses are JSON. Errors follow the ErrorResponse model in models package.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dispatch-email/dispatch/internal/backend"
	"github.com/dispatch-email/dispatch/internal/config"
	"github.com/dispatch-email/dispatch/internal/models"
	"github.com/dispatch-email/dispatch/internal/queue"
	"github.com/dispatch-email/dispatch/internal/store"
	tpl "github.com/dispatch-email/dispatch/internal/template"
	"github.com/dispatch-email/dispatch/internal/token"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	cfg       *config.Config
	store     *store.Store
	backends  *backend.Router
	queue     *queue.Queue
	logger    *slog.Logger
	engine    *tpl.Engine
	tokens    *token.Manager
	startedAt time.Time
}

// New creates a new Server.
func New(cfg *config.Config, store *store.Store, backends *backend.Router, q *queue.Queue, logger *slog.Logger) *Server {
	return &Server{
		cfg:       cfg,
		store:     store,
		backends:  backends,
		queue:     q,
		logger:    logger,
		engine:    tpl.New("shared/templates"),
		tokens:    token.New(cfg.TokenSecret()),
		startedAt: time.Now(),
	}
}

// queueEmail renders a template and enqueues it for delivery.
// Used internally by handlers that need to send emails as a side effect
// (e.g. double opt-in confirmation, welcome emails).
func (s *Server) queueEmail(ctx context.Context, site, toEmail, templateSlug string, data map[string]any) error {
	siteCfg, err := s.cfg.GetSite(site)
	if err != nil {
		return fmt.Errorf("site not found: %w", err)
	}

	// Generate unsubscribe URL
	unsubTok, err := s.tokens.Generate(token.TypeUnsubscribe, toEmail, site, 0)
	if err != nil {
		return fmt.Errorf("generating unsubscribe token: %w", err)
	}
	unsubURL := s.cfg.Server.BaseURL + "/unsubscribe/" + unsubTok

	tplData := &tpl.TemplateData{
		Data: data,
		Site: tpl.SiteInfo{
			Name: siteCfg.Name,
			URL:  s.cfg.Server.BaseURL,
		},
		Subscriber: tpl.SubscriberInfo{
			Email: toEmail,
		},
		UnsubscribeURL: unsubURL,
	}

	rendered, err := s.engine.Render(siteCfg.TemplatesPath(), templateSlug, tplData)
	if err != nil {
		return fmt.Errorf("rendering template %q: %w", templateSlug, err)
	}

	msg := &models.Message{
		Site:            site,
		ToEmail:         toEmail,
		FromEmail:       siteCfg.From,
		FromName:        siteCfg.FromName,
		Template:        templateSlug,
		Subject:         rendered.Subject,
		HTMLBody:        rendered.HTML,
		TextBody:        rendered.Text,
		ListUnsubscribe: unsubURL,
		Backend:         siteCfg.Backend,
	}

	return s.store.CreateMessage(ctx, msg)
}

// Router builds and returns the HTTP router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(s.logMiddleware)

	// Health endpoint (no auth)
	r.Get("/api/v1/health", s.handleHealth)

	// Public endpoints (no auth)
	r.Get("/unsubscribe/{token}", s.handleUnsubscribePage)
	r.Post("/unsubscribe/{token}", s.handleUnsubscribeAction)
	r.Get("/confirm/{token}", s.handleConfirm)

	// Authenticated API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Send endpoints
		r.Post("/sites/{site}/send", s.handleSend)
		r.Post("/sites/{site}/send/raw", s.handleSendRaw)
		r.Post("/sites/{site}/send/batch", s.handleBatchSend)

		// Subscriber endpoints
		r.Post("/sites/{site}/subscribers", s.handleSubscriberCreate)
		r.Get("/sites/{site}/subscribers", s.handleSubscriberList)
		r.Get("/sites/{site}/subscribers/{email}", s.handleSubscriberGet)
		r.Put("/sites/{site}/subscribers/{email}", s.handleSubscriberUpdate)
		r.Delete("/sites/{site}/subscribers/{email}", s.handleSubscriberDelete)

		// Suppression endpoints
		r.Get("/suppressions", s.handleSuppressionList)
		r.Post("/suppressions", s.handleSuppressionAdd)
		r.Delete("/suppressions/{email}", s.handleSuppressionRemove)
		r.Get("/suppressions/check/{email}", s.handleSuppressionCheck)

		// Template endpoints
		r.Get("/sites/{site}/templates", s.handleTemplateList)
		r.Get("/sites/{site}/templates/{slug}", s.handleTemplateGet)
		r.Post("/sites/{site}/templates/{slug}/render", s.handleTemplateRender)

		// Message/audit endpoints
		r.Get("/sites/{site}/messages", s.handleMessageList)
		r.Get("/messages/{id}", s.handleMessageGet)
		r.Get("/messages/{id}/status", s.handleMessageStatus)

		// GDPR endpoints
		r.Post("/gdpr/export", s.handleGDPRExport)
		r.Post("/gdpr/forget", s.handleGDPRForget)

		// Webhook endpoints
		r.Post("/sites/{site}/webhooks", s.handleWebhookCreate)
		r.Get("/sites/{site}/webhooks", s.handleWebhookList)
		r.Delete("/sites/{site}/webhooks/{id}", s.handleWebhookDelete)

		// Auth/key management (master only)
		r.Post("/auth/keys", s.handleKeyCreate)
		r.Get("/auth/keys", s.handleKeyList)
		r.Delete("/auth/keys/{id}", s.handleKeyRevoke)
	})

	return r
}
