// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 MeshManager authors
//
// The redirection protocol and digest-auth handshake in this package are
// derived from MeshCommander (https://github.com/Ylianst/MeshCommander),
// Copyright Ylian Saint-Hilaire, licensed under Apache-2.0. Ported to Go and
// modified by the MeshManager authors.

// Package redirect implements the Intel AMT binary redirection protocol used
// for Serial-over-LAN, KVM and IDE-R over ports 16994 (plain) / 16995 (TLS).
//
// The protocol and digest-auth handshake are reproduced from the reference
// MeshCommander implementation:
//
//	connect → 0x10 StartRedirectionSession → 0x11 reply
//	        → 0x13 AuthenticateSession (query → challenge → digest) → 0x14 replies
//	        → protocol-specific start (SOL: 0x20 settings → 0x21 → 0x27 open)
//
// All multi-byte integers on the wire are little-endian.
package redirect

import (
	"bufio"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Protocol identifiers for StartRedirectionSession.
type Protocol int

const (
	ProtocolSOL  Protocol = 1
	ProtocolKVM  Protocol = 2
	ProtocolIDER Protocol = 3
)

const authURI = "/RedirectionService"

// Target describes how to reach a device's redirection port.
type Target struct {
	Host     string
	Port     int // redirection port (16994/16995), not the WS-MAN port
	TLS      bool
	Insecure bool
	Username string
	Password string
}

// Conn is an authenticated redirection connection. After Connect succeeds the
// caller drives the protocol-specific session (see SOL).
type Conn struct {
	net net.Conn
	r   *bufio.Reader

	wmu sync.Mutex // serializes writes (keystrokes + keepalive)
	seq uint32

	target Target
}

// Connect dials the redirection port, performs the StartRedirectionSession +
// digest authentication handshake for the given protocol, and returns a ready
// connection positioned just past authentication.
func Connect(t Target, proto Protocol) (*Conn, error) {
	addr := net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
	d := net.Dialer{Timeout: 10 * time.Second}

	var nc net.Conn
	var err error
	if t.TLS {
		nc, err = tls.DialWithDialer(&d, "tcp", addr, &tls.Config{InsecureSkipVerify: t.Insecure}) //nolint:gosec // self-signed AMT certs
	} else {
		nc, err = d.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("dial redirection %s: %w", addr, err)
	}

	// Bound the handshake/negotiation phase so a device that accepts the
	// connection but never opens the channel (e.g. SOL disabled) fails fast
	// instead of hanging the HTTP request forever. Cleared once streaming.
	_ = nc.SetDeadline(time.Now().Add(20 * time.Second))

	c := &Conn{net: nc, r: bufio.NewReader(nc), seq: 1, target: t}
	if err := c.handshake(proto); err != nil {
		nc.Close()
		return nil, err
	}
	return c, nil
}

// Close terminates the redirection connection.
func (c *Conn) Close() error { return c.net.Close() }

// ClearDeadline removes the handshake read/write deadline once the session is
// fully open and entering its continuous streaming phase.
func (c *Conn) ClearDeadline() { _ = c.net.SetDeadline(time.Time{}) }

// nextSeq returns and increments the AMT message sequence counter.
func (c *Conn) nextSeq() uint32 {
	s := c.seq
	c.seq++
	return s
}

// write sends raw bytes under the write lock.
func (c *Conn) write(b []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_, err := c.net.Write(b)
	return err
}

func (c *Conn) handshake(proto Protocol) error {
	// 1. StartRedirectionSession with the 4-byte protocol tag.
	var tag []byte
	switch proto {
	case ProtocolSOL:
		tag = []byte{0x10, 0x00, 0x00, 0x00, 'S', 'O', 'L', ' '}
	case ProtocolKVM:
		tag = []byte{0x10, 0x01, 0x00, 0x00, 'K', 'V', 'M', 'R'}
	case ProtocolIDER:
		tag = []byte{0x10, 0x00, 0x00, 0x00, 'I', 'D', 'E', 'R'}
	}
	if err := c.write(tag); err != nil {
		return err
	}

	// 2. Run the auth state machine until the device reports success.
	authed := false
	for !authed {
		cmd, data, err := c.readMessage()
		if err != nil {
			return err
		}
		switch cmd {
		case 0x11: // StartRedirectionSessionReply
			if len(data) < 1 || data[0] != 0 {
				return fmt.Errorf("redirection session refused (status %d)", safeByte(data, 0))
			}
			// Query supported authentication types.
			if err := c.write([]byte{0x13, 0, 0, 0, 0, 0, 0, 0, 0}); err != nil {
				return err
			}
		case 0x14: // AuthenticateSessionReply
			done, err := c.handleAuthReply(data)
			if err != nil {
				return err
			}
			authed = done
		case 0xF0: // session-is-recorded notice
			continue
		default:
			return fmt.Errorf("unexpected redirection command 0x%02x during handshake", cmd)
		}
	}
	return nil
}

// handleAuthReply processes a 0x14 body (already stripped of the command byte).
// data layout: [status, 0, 0, authType, len32(4 bytes), authData...].
func (c *Conn) handleAuthReply(data []byte) (bool, error) {
	if len(data) < 8 {
		return false, fmt.Errorf("short auth reply")
	}
	status := data[0]
	authType := data[3]
	authData := data[8:]

	switch {
	case authType == 0:
		// Query response: authData lists supported auth type bytes.
		if !containsByte(authData, 4) && !containsByte(authData, 3) {
			return false, fmt.Errorf("device does not support digest redirection auth")
		}
		// Kick off digest auth with an initial username-only request.
		return false, c.write(buildAuthMessage(4, c.target.Username, "", "", authURI, "", "", "", ""))

	case (authType == 3 || authType == 4) && status == 1:
		// Challenge: parse realm, nonce, [qop].
		realm, rest := readLP(authData)
		nonce, rest := readLP(rest)
		qop := ""
		if authType == 4 {
			qop, _ = readLP(rest)
		}
		cnonce := randomHex(16)
		const snc = "00000002"
		digest := c.computeDigest(realm, nonce, qop, cnonce, snc)
		if authType == 4 {
			return false, c.write(buildAuthMessage(4, c.target.Username, realm, nonce, authURI, cnonce, snc, digest, qop))
		}
		return false, c.write(buildAuthMessage(3, c.target.Username, realm, nonce, authURI, cnonce, snc, digest))

	case status == 0:
		return true, nil // authenticated

	default:
		return false, fmt.Errorf("redirection authentication failed (status %d, type %d)", status, authType)
	}
}

func (c *Conn) computeDigest(realm, nonce, qop, cnonce, snc string) string {
	ha1 := md5hex(c.target.Username + ":" + realm + ":" + c.target.Password)
	ha2 := md5hex("POST:" + authURI)
	extra := ""
	if qop != "" {
		extra = snc + ":" + cnonce + ":" + qop + ":"
	}
	return md5hex(ha1 + ":" + nonce + ":" + extra + ha2)
}

// readMessage reads exactly one framed redirection message and returns its
// command byte and body (everything after the command byte).
func (c *Conn) readMessage() (byte, []byte, error) {
	var b [1]byte
	if _, err := io.ReadFull(c.r, b[:]); err != nil {
		return 0, nil, err
	}
	cmd := b[0]

	read := func(n int) ([]byte, error) {
		buf := make([]byte, n)
		_, err := io.ReadFull(c.r, buf)
		return buf, err
	}

	switch cmd {
	case 0x11: // StartRedirectionSessionReply: 13 total + oemlen trailer
		head, err := read(12)
		if err != nil {
			return cmd, nil, err
		}
		oemLen := int(head[11])
		oem, err := read(oemLen)
		if err != nil {
			return cmd, nil, err
		}
		return cmd, append(head, oem...), nil
	case 0x14: // AuthenticateSessionReply: 9 header + authDataLen
		head, err := read(8)
		if err != nil {
			return cmd, nil, err
		}
		authDataLen := int(binary.LittleEndian.Uint32(head[4:8]))
		ad, err := read(authDataLen)
		if err != nil {
			return cmd, nil, err
		}
		return cmd, append(head, ad...), nil
	case 0x21: // SOL settings reply: 23 total
		body, err := read(22)
		return cmd, body, err
	case 0x29: // SOL serial settings: 10 total
		body, err := read(9)
		return cmd, body, err
	case 0x2A: // SOL incoming display data: 10 header + payload
		head, err := read(9)
		if err != nil {
			return cmd, nil, err
		}
		dataLen := int(head[7]) | int(head[8])<<8 // bytes 8,9 overall
		payload, err := read(dataLen)
		if err != nil {
			return cmd, nil, err
		}
		return cmd, append(head, payload...), nil
	case 0x2B: // SOL keep-alive: 8 total
		body, err := read(7)
		return cmd, body, err
	case 0x41: // KVM start reply: 8 total
		body, err := read(7)
		return cmd, body, err
	case 0xF0: // session-is-recorded: 1 total
		return cmd, nil, nil
	default:
		return cmd, nil, fmt.Errorf("unknown redirection command 0x%02x", cmd)
	}
}

// --- wire helpers ---

// buildAuthMessage assembles a 0x13 AuthenticateSession message. fields are the
// length-prefixed values in protocol order: user, realm, nonce, uri, cnonce,
// snc, digest, [qop].
func buildAuthMessage(authType byte, fields ...string) []byte {
	var payload []byte
	for _, f := range fields {
		payload = append(payload, byte(len(f)))
		payload = append(payload, f...)
	}
	out := []byte{0x13, 0x00, 0x00, 0x00, authType}
	out = binary.LittleEndian.AppendUint32(out, uint32(len(payload)))
	return append(out, payload...)
}

// readLP reads a single-byte-length-prefixed string, returning it and the rest.
func readLP(b []byte) (string, []byte) {
	if len(b) == 0 {
		return "", b
	}
	n := int(b[0])
	if 1+n > len(b) {
		n = len(b) - 1
	}
	return string(b[1 : 1+n]), b[1+n:]
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s)) //nolint:gosec // AMT digest auth mandates MD5
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func containsByte(b []byte, v byte) bool { return strings.IndexByte(string(b), v) >= 0 }

func safeByte(b []byte, i int) byte {
	if i < len(b) {
		return b[i]
	}
	return 0xff
}
