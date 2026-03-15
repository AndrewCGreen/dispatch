package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dispatch-email/dispatch/internal/models"
	tpl "github.com/dispatch-email/dispatch/internal/template"
)

// handleSend processes POST /api/v1/sites/{site}/send
func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
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

	var req models.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.To == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Field 'to' is required")
		return
	}
	if req.Template == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Field 'template' is required")
		return
	}

	// Check suppression
	suppressed, err := s.store.IsSuppressed(r.Context(), req.To)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Failed to check suppression")
		return
	}
	if suppressed {
		writeError(w, http.StatusConflict, "RECIPIENT_SUPPRESSED", "Recipient is on the suppression list")
		return
	}

	// Render template
	data := &tpl.TemplateData{
		Data: req.Data,
		Site: tpl.SiteInfo{
			Name: siteCfg.Name,
		},
		Subscriber: tpl.SubscriberInfo{
			Email: req.To,
		},
	}

	rendered, err := s.engine.Render(siteCfg.TemplatesPath(), req.Template, data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "TEMPLATE_ERROR", err.Error())
		return
	}

	// Create message record
	tags, _ := json.Marshal(req.Tags)
	meta, _ := json.Marshal(req.Metadata)

	msg := &models.Message{
		Site:     site,
		ToEmail:  req.To,
		Template: req.Template,
		Subject:  rendered.Subject,
		Backend:  siteCfg.Backend,
		Tags:     tags,
		Metadata: meta,
	}

	if err := s.store.CreateMessage(r.Context(), msg); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Failed to queue message")
		return
	}

	resp := models.SendResponse{
		ID:       msg.ID,
		Status:   "queued",
		To:       req.To,
		Template: req.Template,
		Site:     site,
		QueuedAt: msg.QueuedAt,
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// handleSendRaw processes POST /api/v1/sites/{site}/send/raw
func (s *Server) handleSendRaw(w http.ResponseWriter, r *http.Request) {
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

	var req models.SendRawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.To == "" || req.Subject == "" || req.HTML == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Fields 'to', 'subject', and 'html' are required")
		return
	}

	// Check suppression
	suppressed, _ := s.store.IsSuppressed(r.Context(), req.To)
	if suppressed {
		writeError(w, http.StatusConflict, "RECIPIENT_SUPPRESSED", "Recipient is on the suppression list")
		return
	}

	// Render inline content with data substitution
	data := &tpl.TemplateData{
		Data: req.Data,
		Site: tpl.SiteInfo{Name: siteCfg.Name},
	}
	rendered, err := s.engine.RenderRaw(req.Subject, req.HTML, req.Text, data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "RENDER_ERROR", err.Error())
		return
	}

	_ = rendered // Will be stored/used by queue worker

	msg := &models.Message{
		Site:    site,
		ToEmail: req.To,
		Subject: rendered.Subject,
		Backend: siteCfg.Backend,
	}

	if err := s.store.CreateMessage(r.Context(), msg); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Failed to queue message")
		return
	}

	writeJSON(w, http.StatusAccepted, models.SendResponse{
		ID:       msg.ID,
		Status:   "queued",
		To:       req.To,
		Site:     site,
		QueuedAt: msg.QueuedAt,
	})
}

// handleBatchSend processes POST /api/v1/sites/{site}/send/batch
func (s *Server) handleBatchSend(w http.ResponseWriter, r *http.Request) {
	site := chi.URLParam(r, "site")
	if !requireSiteAccess(r, site) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "No access to this site")
		return
	}

	_, err := s.cfg.GetSite(site)
	if err != nil {
		writeError(w, http.StatusNotFound, "SITE_NOT_FOUND", err.Error())
		return
	}

	var req models.BatchSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Template == "" || len(req.Recipients) == 0 {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Fields 'template' and 'recipients' are required")
		return
	}

	queued := 0
	suppressed := 0

	for _, recip := range req.Recipients {
		isSup, _ := s.store.IsSuppressed(r.Context(), recip.To)
		if isSup {
			suppressed++
			continue
		}

		msg := &models.Message{
			Site:     site,
			ToEmail:  recip.To,
			Template: req.Template,
		}
		if err := s.store.CreateMessage(r.Context(), msg); err != nil {
			continue
		}
		queued++
	}

	writeJSON(w, http.StatusAccepted, models.BatchSendResponse{
		BatchID:    "batch_" + site, // TODO: generate proper batch ID
		Total:      len(req.Recipients),
		Queued:     queued,
		Suppressed: suppressed,
	})
}
