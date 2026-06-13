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

// handleKVM bridges a browser WebSocket to a KVM session. After the redirection
// handshake the channel is a raw RFB byte pipe in both directions; the AMT RFB
// client logic lives in the frontend.
func (s *Server) handleKVM(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)

	// Ensure KVM is enabled in firmware first; a disabled SAP yields a black
	// screen with no error. Best-effort — don't block connect if it fails.
	if err := sess.EnableKVM(); err != nil {
		slog.Warn("enable KVM failed (continuing)", "err", err.Error())
	} else {
		slog.Info("KVM redirection enabled")
	}

	conn, err := redirect.StartKVM(sess.RedirectionTarget())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		conn.Close()
		return
	}
	defer ws.Close()
	defer conn.Close()

	// Device → browser: stream raw RFB bytes.
	go func() {
		buf := make([]byte, 16384)
		rd := conn.Reader()
		for {
			n, err := rd.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				ws.Close()
				return
			}
		}
	}()

	// Browser → device: forward RFB client messages verbatim.
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if err := conn.RawWrite(msg); err != nil {
			slog.Debug("kvm write failed", "err", err)
			return
		}
	}
}
