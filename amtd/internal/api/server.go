// Package api exposes the AMT session manager over a local HTTP + WebSocket API.
package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/emaspa/meshmanager/amtd/internal/amt"
)

// Server wires the AMT session manager to HTTP routes.
type Server struct {
	sessions *amt.SessionManager
	token    string
	version  string
}

// NewServer constructs a Server. token, when non-empty, must be presented as a
// Bearer token on every request.
func NewServer(sessions *amt.SessionManager, token, version string) *Server {
	return &Server{sessions: sessions, token: token, version: version}
}

// Router builds the chi router with all routes and middleware.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(corsMiddleware)
	r.Use(s.authMiddleware)

	r.Get("/api/health", s.handleHealth)
	r.Post("/api/log", s.handleClientLog)
	r.Post("/api/connect", s.handleConnect)
	r.Post("/api/discover", s.handleDiscover)
	r.Get("/api/devices", s.handleListDevices)

	r.Route("/api/devices/{id}", func(r chi.Router) {
		r.Use(s.deviceCtx)
		r.Post("/disconnect", s.handleDisconnect)
		r.Get("/info", s.handleInfo)
		r.Get("/power", s.handlePowerState)
		r.Post("/power", s.handlePowerAction)
		r.Post("/boot", s.handleBoot)
		r.Get("/hardware", s.handleHardware)
		r.Get("/network", s.handleNetwork)
		r.Get("/eventlog", s.handleEventLog)
		r.Get("/auditlog", s.handleAuditLog)
		r.Post("/ider/start", s.handleIDERStart)
		r.Post("/ider/stop", s.handleIDERStop)
		r.Get("/ider/status", s.handleIDERStatus)
		r.Get("/sol", s.handleSOL)   // WebSocket
		r.Get("/kvm", s.handleKVM)   // WebSocket
	})

	return r
}

// authMiddleware enforces the bearer token when one is configured. Health is
// always allowed so the launcher can probe readiness.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		// WebSocket upgrades can't easily set Authorization headers from the
		// browser, so also accept the token via the access_token query param.
		got := bearerToken(r)
		if got == "" {
			got = r.URL.Query().Get("access_token")
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid or missing token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware reflects local-origin requests so the Vite dev server and the
// Tauri webview (both cross-origin to 127.0.0.1) can call the API. Only
// localhost / 127.0.0.1 / tauri origins are allowed; the daemon binds to
// loopback so this is not a public surface.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, p := range []string{
		"http://localhost", "https://localhost",
		"http://127.0.0.1", "https://127.0.0.1",
		"http://tauri.localhost", "https://tauri.localhost",
		"tauri://localhost",
	} {
		if strings.HasPrefix(origin, p) {
			return true
		}
	}
	return false
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": s.version})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
