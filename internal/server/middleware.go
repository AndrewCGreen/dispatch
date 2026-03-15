package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	ctxKeyType contextKey = "key_type"
	ctxSite    contextKey = "key_site"
)

// logMiddleware logs each request.
func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// authMiddleware validates API keys.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing Authorization header")
			return
		}

		key := strings.TrimPrefix(auth, "Bearer ")
		if key == auth {
			// Also support Basic auth
			key = strings.TrimPrefix(auth, "Basic ")
		}

		// Check master key first
		if s.cfg.Auth.MasterKey != "" && key == s.cfg.Auth.MasterKey {
			ctx := context.WithValue(r.Context(), ctxKeyType, "master")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Check site-specific keys
		for slug, site := range s.cfg.Sites {
			if site.APIKey != "" && key == site.APIKey {
				ctx := context.WithValue(r.Context(), ctxKeyType, "site")
				ctx = context.WithValue(ctx, ctxSite, slug)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// TODO: check database-stored API keys (hashed)

		writeError(w, http.StatusUnauthorized, "INVALID_KEY", "Invalid API key")
	})
}

// requireSiteAccess checks that the authenticated key has access to the requested site.
func requireSiteAccess(r *http.Request, site string) bool {
	keyType, _ := r.Context().Value(ctxKeyType).(string)
	if keyType == "master" {
		return true
	}
	keySite, _ := r.Context().Value(ctxSite).(string)
	return keySite == site
}
