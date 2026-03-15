package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/dispatch-email/dispatch/internal/models"
	"github.com/dispatch-email/dispatch/internal/token"
)

func (s *Server) handleSubscriberCreate(w http.ResponseWriter, r *http.Request) {
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

	var req models.SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Field 'email' is required")
		return
	}

	// Check suppression
	suppressed, _ := s.store.IsSuppressed(r.Context(), req.Email)
	if suppressed {
		writeError(w, http.StatusConflict, "RECIPIENT_SUPPRESSED", "Email is on the suppression list")
		return
	}

	// Check if already exists
	existing, _ := s.store.GetSubscriber(r.Context(), site, req.Email)
	if existing != nil {
		writeError(w, http.StatusConflict, "ALREADY_EXISTS", "Subscriber already exists")
		return
	}

	// Determine initial status
	status := models.StatusActive
	if siteCfg.DoubleOptin {
		status = models.StatusPending
	}

	attrs, _ := json.Marshal(req.Attributes)

	sub := &models.Subscriber{
		Site:       site,
		Email:      req.Email,
		Name:       req.Name,
		Status:     status,
		Attributes: attrs,
		Lists:      req.Lists,
	}

	if err := s.store.CreateSubscriber(r.Context(), sub); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Failed to create subscriber")
		return
	}

	// Log consent
	if req.Consent != nil && s.cfg.Compliance.ConsentLogging {
		s.store.LogConsent(r.Context(), &models.ConsentRecord{
			Email:  req.Email,
			Site:   site,
			Action: models.ConsentSubscribe,
			Source: req.Consent.Source,
			IP:     req.Consent.IP,
			URL:    req.Consent.URL,
		})
	}

	confirmSent := false
	if siteCfg.DoubleOptin && siteCfg.OptinTpl != "" {
		// Generate a confirmation token with a 72-hour TTL
		confirmTok, err := s.tokens.Generate(token.TypeConfirm, req.Email, site, 72*time.Hour)
		if err != nil {
			slog.Error("failed to generate confirm token", "email", req.Email, "error", err)
		} else {
			confirmURL := s.cfg.Server.BaseURL + "/confirm/" + confirmTok
			err = s.queueEmail(r.Context(), site, req.Email, siteCfg.OptinTpl, map[string]any{
				"confirm_url": confirmURL,
				"name":        req.Name,
			})
			if err != nil {
				slog.Error("failed to queue optin email", "email", req.Email, "error", err)
			} else {
				confirmSent = true
			}
		}
	}

	resp := map[string]any{
		"email":  sub.Email,
		"status": sub.Status,
	}
	if siteCfg.DoubleOptin {
		resp["confirm_sent"] = confirmSent
		resp["message"] = "Double opt-in confirmation sent"
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleSubscriberList(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	// TODO: implement pagination and listing
	writeJSON(w, http.StatusOK, map[string]any{
		"subscribers": []any{},
		"total":       0,
		"page":        1,
		"per_page":    50,
		"has_more":    false,
	})
}

func (s *Server) handleSubscriberGet(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	email := chi.URLParam(r, "email")

	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	sub, err := s.store.GetSubscriber(r.Context(), site, email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Database error")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Subscriber not found")
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

func (s *Server) handleSubscriberUpdate(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	// TODO: implement subscriber update
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleSubscriberDelete(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	email := chi.URLParam(r, "email")

	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	if err := s.store.DeleteSubscriber(r.Context(), site, email); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	// Log consent
	if s.cfg.Compliance.ConsentLogging {
		s.store.LogConsent(r.Context(), &models.ConsentRecord{
			Email:  email,
			Site:   site,
			Action: models.ConsentUnsubscribe,
			Source: "api",
			IP:     r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
