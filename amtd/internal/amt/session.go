package amt

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"

	"github.com/emaspa/meshmanager/amtd/internal/redirect"
)

// DefaultPort is the standard Intel AMT WS-MAN port (non-TLS). 16993 is TLS.
const (
	DefaultPort    = 16992
	DefaultTLSPort = 16993
)

// ConnectParams describes how to reach and authenticate to an AMT device.
type ConnectParams struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
	// Insecure skips TLS certificate verification (self-signed AMT certs).
	Insecure bool `json:"insecure"`
	// Name is an optional friendly label for the device.
	Name string `json:"name"`
}

// Session is a live, authenticated connection to a single AMT device.
//
// The embedded wsman.Messages holds the credentials and HTTP client, so it is
// reused across requests. AMT firmware tolerates concurrent WS-MAN calls
// poorly, so every call is serialized through mu.
type Session struct {
	ID        string        `json:"id"`
	Host      string        `json:"host"`
	Port      int           `json:"port"`
	Name      string        `json:"name"`
	TLS       bool          `json:"tls"`
	Username  string        `json:"username"`
	CreatedAt time.Time     `json:"createdAt"`

	mu  sync.Mutex      `json:"-"`
	wsman wsman.Messages `json:"-"`
	params ConnectParams `json:"-"`
}

// RedirectionTarget returns the connection details for the binary redirection
// port (SOL/KVM/IDE-R), which lives 2 ports above the WS-MAN port by
// convention (16994/16995 vs 16992/16993).
func (s *Session) RedirectionTarget() redirect.Target {
	return redirect.Target{
		Host:     s.Host,
		Port:     s.Port + 2,
		TLS:      s.TLS,
		Insecure: s.params.Insecure,
		Username: s.Username,
		Password: s.params.Password,
	}
}

// withWSMAN runs fn against the session's WS-MAN client under the per-session
// lock so AMT firmware never sees overlapping requests on one connection.
func (s *Session) withWSMAN(fn func(m *wsman.Messages) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(&s.wsman)
}

// SessionManager owns the set of active device connections.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	debug    bool
}

// NewSessionManager creates an empty manager.
func NewSessionManager(debug bool) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		debug:    debug,
	}
}

// Connect opens and verifies a new session to an AMT device. It returns the
// new session; an error means the host was unreachable or auth failed.
func (m *SessionManager) Connect(p ConnectParams) (*Session, error) {
	if p.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if p.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if p.Port == 0 {
		if p.TLS {
			p.Port = DefaultTLSPort
		} else {
			p.Port = DefaultPort
		}
	}

	cp := client.Parameters{
		Target:            p.Host,
		Username:          p.Username,
		Password:          p.Password,
		UseDigest:         true,
		UseTLS:            p.TLS,
		SelfSignedAllowed: p.Insecure,
		LogAMTMessages:    m.debug,
	}
	// The library defaults to the standard ports; when a custom port is given
	// we pass host:port as the target.
	if (p.TLS && p.Port != DefaultTLSPort) || (!p.TLS && p.Port != DefaultPort) {
		cp.Target = fmt.Sprintf("%s:%d", p.Host, p.Port)
	}

	messages := wsman.NewMessages(cp)

	sess := &Session{
		ID:        newID(),
		Host:      p.Host,
		Port:      p.Port,
		Name:      p.Name,
		TLS:       p.TLS,
		Username:  p.Username,
		CreatedAt: time.Now(),
		wsman:     messages,
		params:    p,
	}

	// Verify reachability + credentials with a cheap authenticated call.
	if err := sess.withWSMAN(func(msg *wsman.Messages) error {
		_, err := msg.AMT.SetupAndConfigurationService.GetUUID()
		return err
	}); err != nil {
		return nil, fmt.Errorf("connect to %s: %w", p.Host, err)
	}

	m.mu.Lock()
	m.sessions[sess.ID] = sess
	m.mu.Unlock()
	return sess, nil
}

// Get returns the session with the given id, or false if not connected.
func (m *SessionManager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// List returns all active sessions.
func (m *SessionManager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

// Disconnect removes a session. Returns false if it did not exist.
func (m *SessionManager) Disconnect(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; !ok {
		return false
	}
	delete(m.sessions, id)
	return true
}

// CloseAll drops every session (used on shutdown).
func (m *SessionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*Session)
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
