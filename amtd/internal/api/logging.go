package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder captures the response status and size for request logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// requestLogger logs one line per HTTP request. Health checks log at debug to
// avoid spamming the file; failures (>=400) log at warn/error.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't wrap WebSocket upgrades — the hijacker must reach the real writer.
		if r.Header.Get("Upgrade") != "" {
			slog.Info("ws", "path", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		dur := time.Since(start)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"ms", dur.Milliseconds(),
		}
		switch {
		case rec.status >= 500:
			slog.Error("request", attrs...)
		case rec.status >= 400:
			slog.Warn("request", attrs...)
		case r.URL.Path == "/api/health":
			slog.Debug("request", attrs...)
		default:
			slog.Info("request", attrs...)
		}
	})
}

// handleClientLog ingests a log record from the frontend so UI errors land in
// the same file as the sidecar's, making tester bug reports self-contained.
func (s *Server) handleClientLog(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Level   string `json:"level"`
		Message string `json:"message"`
		Context string `json:"context"`
		Stack   string `json:"stack"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid log record")
		return
	}
	level := slog.LevelInfo
	switch body.Level {
	case "error":
		level = slog.LevelError
	case "warn":
		level = slog.LevelWarn
	case "debug":
		level = slog.LevelDebug
	}
	attrs := []any{"src", "ui"}
	if body.Context != "" {
		attrs = append(attrs, "context", body.Context)
	}
	if body.Stack != "" {
		attrs = append(attrs, "stack", body.Stack)
	}
	slog.Log(r.Context(), level, body.Message, attrs...)
	w.WriteHeader(http.StatusNoContent)
}
