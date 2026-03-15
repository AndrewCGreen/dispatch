package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/dispatch-email/dispatch/internal/models"
	tpl "github.com/dispatch-email/dispatch/internal/template"
	"github.com/dispatch-email/dispatch/internal/token"
)

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	pending, processing, failed, _ := s.store.QueueStats(r.Context())
	backendHealth := s.backends.HealthAll(r.Context())

	dbStatus := "ok"
	if _, err := s.store.IsSuppressed(r.Context(), "healthcheck@dispatch.local"); err != nil {
		dbStatus = fmt.Sprintf("error: %v", err)
	}

	resp := models.HealthResponse{
		Status:   "healthy",
		Version:  "0.1.0-dev",
		Uptime:   time.Since(s.startedAt).Round(time.Second).String(),
		Database: dbStatus,
		Queue: models.QueueHealth{
			Pending:    pending,
			Processing: processing,
			Failed:     failed,
		},
		Backends: backendHealth,
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Unsubscribe ---

func (s *Server) handleUnsubscribePage(w http.ResponseWriter, r *http.Request) {
	tok := chi.URLParam(r, "token")
	claims, err := s.tokens.Verify(tok)
	if err != nil || claims.Type != token.TypeUnsubscribe {
		writeHTMLPage(w, http.StatusBadRequest, "Invalid Link",
			"<h1>Invalid or expired link</h1><p>This unsubscribe link is not valid.</p>")
		return
	}

	writeHTMLPage(w, http.StatusOK, "Unsubscribe", fmt.Sprintf(`
<h1>Unsubscribe</h1>
<p>Click below to unsubscribe <strong>%s</strong> from future emails.</p>
<form method="POST">
  <button type="submit" style="padding:12px 24px;background:#dc2626;color:#fff;border:none;border-radius:6px;cursor:pointer;font-size:16px;">
    Unsubscribe
  </button>
</form>`, claims.Email))
}

func (s *Server) handleUnsubscribeAction(w http.ResponseWriter, r *http.Request) {
	tok := chi.URLParam(r, "token")
	claims, err := s.tokens.Verify(tok)
	if err != nil || claims.Type != token.TypeUnsubscribe {
		writeHTMLPage(w, http.StatusBadRequest, "Invalid Link",
			"<h1>Invalid or expired link</h1><p>This unsubscribe link is not valid.</p>")
		return
	}

	ctx := r.Context()

	// Mark subscriber as unsubscribed (best-effort — may not exist as a subscriber)
	s.store.UnsubscribeSubscriber(ctx, claims.Site, claims.Email)

	// Add to global suppression
	s.store.AddSuppression(ctx, &models.Suppression{
		Email:  claims.Email,
		Reason: models.ReasonUnsubscribe,
		Note:   fmt.Sprintf("Unsubscribed via link from site %s", claims.Site),
	})

	// Log consent
	if s.cfg.Compliance.ConsentLogging {
		s.store.LogConsent(ctx, &models.ConsentRecord{
			Email:  claims.Email,
			Site:   claims.Site,
			Action: models.ConsentUnsubscribe,
			Source: "unsubscribe-link",
			IP:     r.RemoteAddr,
		})
	}

	writeHTMLPage(w, http.StatusOK, "Unsubscribed", fmt.Sprintf(`
<h1>You've been unsubscribed</h1>
<p><strong>%s</strong> has been removed from our mailing list.</p>
<p>You won't receive any more emails from us.</p>`, claims.Email))
}

// --- Double Opt-In Confirmation ---

func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	tok := chi.URLParam(r, "token")
	claims, err := s.tokens.Verify(tok)
	if err != nil || claims.Type != token.TypeConfirm {
		writeHTMLPage(w, http.StatusBadRequest, "Invalid Link",
			"<h1>Invalid or expired link</h1><p>This confirmation link is not valid or has expired. Please subscribe again to receive a new one.</p>")
		return
	}

	ctx := r.Context()

	if err := s.store.ConfirmSubscriber(ctx, claims.Site, claims.Email); err != nil {
		// Already confirmed is not an error worth showing as a failure
		writeHTMLPage(w, http.StatusOK, "Already Confirmed",
			fmt.Sprintf("<h1>Already confirmed</h1><p><strong>%s</strong> is already subscribed and active.</p>", claims.Email))
		return
	}

	// Log consent confirmation
	if s.cfg.Compliance.ConsentLogging {
		s.store.LogConsent(ctx, &models.ConsentRecord{
			Email:  claims.Email,
			Site:   claims.Site,
			Action: models.ConsentSubscribe,
			Source: "double-optin-confirm",
			IP:     r.RemoteAddr,
		})
	}

	// Send welcome email if configured
	siteCfg, _ := s.cfg.GetSite(claims.Site)
	if siteCfg != nil && siteCfg.WelcomeTpl != "" {
		if err := s.queueEmail(ctx, claims.Site, claims.Email, siteCfg.WelcomeTpl, map[string]any{
			"name": "",
		}); err != nil {
			slog.Error("failed to queue welcome email", "email", claims.Email, "error", err)
		}
	}

	writeHTMLPage(w, http.StatusOK, "Confirmed!", fmt.Sprintf(`
<h1>You're confirmed!</h1>
<p><strong>%s</strong> has been successfully subscribed.</p>
<p>Thanks for confirming your email address.</p>`, claims.Email))
}

// --- Templates ---

func (s *Server) handleTemplateList(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	siteCfg, err := s.cfg.GetSite(site)
	if err != nil {
		writeError(w, http.StatusNotFound, "SITE_NOT_FOUND", err.Error())
		return
	}

	tplDir := siteCfg.TemplatesPath()
	entries, err := os.ReadDir(tplDir)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"templates": []any{}})
		return
	}

	var templates []map[string]string
	for _, e := range entries {
		name := e.Name()
		if name[0] == '_' { // skip base templates
			continue
		}
		slug := name
		if filepath.Ext(name) == ".html" {
			slug = name[:len(name)-5]
		}
		templates = append(templates, map[string]string{
			"slug": slug,
			"type": func() string {
				if e.IsDir() {
					return "directory"
				}
				return "single"
			}(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func (s *Server) handleTemplateGet(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	slug := chi.URLParam(r, "slug")

	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	siteCfg, err := s.cfg.GetSite(site)
	if err != nil {
		writeError(w, http.StatusNotFound, "SITE_NOT_FOUND", err.Error())
		return
	}

	tplPath := filepath.Join(siteCfg.TemplatesPath(), slug+".html")
	content, err := os.ReadFile(tplPath)
	if err != nil {
		// Try directory
		tplPath = filepath.Join(siteCfg.TemplatesPath(), slug, "body.html")
		content, err = os.ReadFile(tplPath)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Template not found")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"slug":   slug,
		"source": string(content),
	})
}

func (s *Server) handleTemplateRender(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	slug := chi.URLParam(r, "slug")

	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	siteCfg, err := s.cfg.GetSite(site)
	if err != nil {
		writeError(w, http.StatusNotFound, "SITE_NOT_FOUND", err.Error())
		return
	}

	var req struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	data := &tpl.TemplateData{
		Data: req.Data,
		Site: tpl.SiteInfo{Name: siteCfg.Name},
	}

	rendered, err := s.engine.Render(siteCfg.TemplatesPath(), slug, data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "RENDER_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"subject": rendered.Subject,
		"html":    rendered.HTML,
		"text":    rendered.Text,
	})
}

// --- Messages ---

func (s *Server) handleMessageList(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}
	// TODO: implement message listing with pagination
	writeJSON(w, http.StatusOK, map[string]any{
		"messages": []any{},
		"total":    0,
	})
}

func (s *Server) handleMessageGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	msg, err := s.store.GetMessage(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Database error")
		return
	}
	if msg == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

func (s *Server) handleMessageStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	msg, err := s.store.GetMessage(r.Context(), id)
	if err != nil || msg == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     msg.ID,
		"status": msg.Status,
	})
}

// --- GDPR ---

func (s *Server) handleGDPRExport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Field 'email' is required")
		return
	}

	// Collect data across all sites
	export := map[string]any{
		"email": req.Email,
		"sites": map[string]any{},
	}

	sites := export["sites"].(map[string]any)
	for slug := range s.cfg.Sites {
		sub, _ := s.store.GetSubscriber(r.Context(), slug, req.Email)
		if sub != nil {
			sites[slug] = sub
		}
	}

	// Log the export action
	if s.cfg.Compliance.ConsentLogging {
		s.store.LogConsent(r.Context(), &models.ConsentRecord{
			Email:  req.Email,
			Action: models.ConsentExport,
			Source: "api",
			IP:     r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, export)
}

func (s *Server) handleGDPRForget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		Confirm bool   `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Field 'email' is required")
		return
	}
	if !req.Confirm {
		writeError(w, http.StatusBadRequest, "CONFIRMATION_REQUIRED", "Set 'confirm': true to proceed")
		return
	}

	// Delete subscriber from all sites
	for slug := range s.cfg.Sites {
		s.store.DeleteSubscriber(r.Context(), slug, req.Email)
	}

	// Add to global suppression
	s.store.AddSuppression(r.Context(), &models.Suppression{
		Email:  req.Email,
		Reason: models.ReasonGDPR,
		Note:   "Right to erasure request",
	})

	// Log the forget action
	if s.cfg.Compliance.ConsentLogging {
		s.store.LogConsent(r.Context(), &models.ConsentRecord{
			Email:  req.Email,
			Action: models.ConsentForget,
			Source: "api",
			IP:     r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"email":  req.Email,
		"status": "forgotten",
	})
}

// --- Webhooks ---

func (s *Server) handleWebhookCreate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "coming in v0.2"})
}

func (s *Server) handleWebhookList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"webhooks": []any{}})
}

func (s *Server) handleWebhookDelete(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Auth/Keys ---

func (s *Server) handleKeyCreate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "coming in v0.2"})
}

func (s *Server) handleKeyList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"keys": []any{}})
}

func (s *Server) handleKeyRevoke(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorResponse{
		Error:   code,
		Message: message,
		Code:    code,
	})
}

func writeHTMLPage(w http.ResponseWriter, status int, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>%s</title></head>
<body style="font-family:sans-serif;max-width:500px;margin:50px auto;text-align:center;">
%s
</body></html>`, title, body)
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
