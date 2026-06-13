package api

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/emaspa/meshmanager/amtd/internal/redirect"
)

// upgrader accepts WebSocket upgrades. Origin checks are handled by the CORS
// middleware + bearer token; the daemon only binds to loopback.
var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// handleSOL bridges a browser WebSocket to a Serial-over-LAN session: device
// output is relayed as binary WS frames; inbound WS frames are sent as serial
// input/keystrokes.
func (s *Server) handleSOL(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)

	sol, err := redirect.StartSOL(sess.RedirectionTarget())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sol.Close()
		return
	}
	defer ws.Close()
	defer sol.Close()

	// Device → browser.
	go func() {
		for data := range sol.Output() {
			if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		}
		// SOL ended; close the socket so the read loop below unblocks.
		ws.Close()
	}()

	// Browser → device.
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if err := sol.Write(msg); err != nil {
			slog.Debug("sol write failed", "err", err)
			return
		}
	}
}

func (s *Server) handleKVM(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "KVM not yet implemented")
}
