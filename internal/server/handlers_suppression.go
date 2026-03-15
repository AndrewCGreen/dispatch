package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dispatch-email/dispatch/internal/models"
)

func (s *Server) handleSuppressionList(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 50)

	sups, total, err := s.store.ListSuppressions(r.Context(), page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Database error")
		return
	}

	writeJSON(w, http.StatusOK, models.PaginatedResponse{
		Data:    sups,
		Total:   total,
		Page:    page,
		PerPage: perPage,
		HasMore: (page * perPage) < total,
	})
}

func (s *Server) handleSuppressionAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email  string `json:"email"`
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "Field 'email' is required")
		return
	}

	reason := models.SuppressionReason(req.Reason)
	if reason == "" {
		reason = models.ReasonManual
	}

	sup := &models.Suppression{
		Email:  req.Email,
		Reason: reason,
		Note:   req.Note,
	}

	if err := s.store.AddSuppression(r.Context(), sup); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Failed to add suppression")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"email":  req.Email,
		"status": "suppressed",
	})
}

func (s *Server) handleSuppressionRemove(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	if err := s.store.RemoveSuppression(r.Context(), email); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Failed to remove suppression")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"email":  email,
		"status": "removed",
	})
}

func (s *Server) handleSuppressionCheck(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	suppressed, err := s.store.IsSuppressed(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"email":      email,
		"suppressed": suppressed,
	})
}
