package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/emaspa/meshmanager/amtd/internal/amt"
)

type ctxKey string

const sessionKey ctxKey = "session"

// deviceCtx resolves {id} to a live session and stashes it on the request.
func (s *Server) deviceCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		sess, ok := s.sessions.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "device not connected")
			return
		}
		ctx := context.WithValue(r.Context(), sessionKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sessionFrom(r *http.Request) *amt.Session {
	return r.Context().Value(sessionKey).(*amt.Session)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	var p amt.ConnectParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sess, err := s.sessions.Connect(p)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.sessions.List())
}

func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CIDR string `json:"cidr"`
		Port int    `json:"port"`
		TLS  bool   `json:"tls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	found, err := amt.Scan(ctx, body.CIDR, body.Port, body.TLS, 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.sessions.Disconnect(id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	info, err := sessionFrom(r).Info()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handlePowerState(w http.ResponseWriter, r *http.Request) {
	ps, err := sessionFrom(r).PowerState()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

func (s *Server) handlePowerAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ret, err := sessionFrom(r).Power(body.Action)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "returnValue": ret})
}

func (s *Server) handleBoot(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Device string `json:"device"`
		Power  string `json:"power"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := sessionFrom(r).Boot(body.Device, body.Power); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleHardware(w http.ResponseWriter, r *http.Request) {
	hw, err := sessionFrom(r).Hardware()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hw)
}

func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request) {
	nics, err := sessionFrom(r).Network()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nics)
}

func (s *Server) handleEventLog(w http.ResponseWriter, r *http.Request) {
	log, err := sessionFrom(r).EventLog()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, log)
}

func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	log, err := sessionFrom(r).AuditLog()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, log)
}
