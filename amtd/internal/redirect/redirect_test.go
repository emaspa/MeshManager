package redirect

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildAuthMessage_InitialRequest(t *testing.T) {
	// The username-only request that triggers the digest challenge, matching
	// the reference console's type-4 query form.
	got := buildAuthMessage(4, "admin", "", "", authURI, "", "", "", "")

	var want []byte
	want = append(want, 0x13, 0x00, 0x00, 0x00, 0x04)
	// payload = lp(admin) lp() lp() lp(authURI) lp() lp() lp() lp()
	payload := []byte{0x05}
	payload = append(payload, "admin"...)
	payload = append(payload, 0x00, 0x00) // empty realm, nonce
	payload = append(payload, byte(len(authURI)))
	payload = append(payload, authURI...)
	payload = append(payload, 0x00, 0x00, 0x00, 0x00) // empty cnonce, snc, digest, qop
	want = binary.LittleEndian.AppendUint32(want, uint32(len(payload)))
	want = append(want, payload...)

	if !bytes.Equal(got, want) {
		t.Fatalf("auth message mismatch:\n got=%v\nwant=%v", got, want)
	}
	// Sanity: payload length must equal user+uri+8 (8 length-prefix bytes).
	if int(binary.LittleEndian.Uint32(got[5:9])) != len("admin")+len(authURI)+8 {
		t.Fatalf("payload length field wrong")
	}
}

func TestReadLP(t *testing.T) {
	buf := []byte{0x03, 'a', 'b', 'c', 0x02, 'x', 'y'}
	s1, rest := readLP(buf)
	s2, rest := readLP(rest)
	if s1 != "abc" || s2 != "xy" || len(rest) != 0 {
		t.Fatalf("readLP got %q %q rest=%v", s1, s2, rest)
	}
}

func TestReadMessage_DisplayData(t *testing.T) {
	// 0x2A frame: 10-byte header (datalen little-endian at offset 8,9) + payload.
	frame := []byte{0x2A, 0, 0, 0, 0, 0, 0, 0}
	frame = append(frame, 0x05, 0x00) // datalen = 5 at offsets 8,9
	frame = append(frame, []byte("hello")...)

	c := &Conn{r: bufio.NewReader(bytes.NewReader(frame))}
	cmd, data, err := c.readMessage()
	if err != nil {
		t.Fatal(err)
	}
	if cmd != 0x2A {
		t.Fatalf("cmd = 0x%02x", cmd)
	}
	// readLoop extracts payload as data[9:].
	if got := string(data[9:]); got != "hello" {
		t.Fatalf("payload = %q", got)
	}
}

func TestReadMessage_AuthReply(t *testing.T) {
	// 0x14: header [status,0,0,authType, len32], then authData.
	authData := []byte{0x04, 0x03} // supported types list
	frame := []byte{0x14, 0x01, 0x00, 0x00, 0x00}
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(authData)))
	frame = append(frame, authData...)

	c := &Conn{r: bufio.NewReader(bytes.NewReader(frame))}
	cmd, data, err := c.readMessage()
	if err != nil {
		t.Fatal(err)
	}
	if cmd != 0x14 {
		t.Fatalf("cmd = 0x%02x", cmd)
	}
	if data[0] != 0x01 { // status
		t.Fatalf("status = %d", data[0])
	}
	if !bytes.Equal(data[8:], authData) {
		t.Fatalf("authData = %v", data[8:])
	}
}

func TestComputeDigest_Stable(t *testing.T) {
	c := &Conn{target: Target{Username: "admin", Password: "P@ssw0rd"}}
	// Regression pin: digest of a fixed challenge. Computed once from the
	// RFC2617-style formula this implementation uses; guards against
	// accidental changes to field ordering or separators.
	got := c.computeDigest("Digest:abc", "nonce123", "auth", "cnonceXYZ", "00000002")
	if len(got) != 32 {
		t.Fatalf("digest length = %d, want 32 hex chars", len(got))
	}
	// Recompute independently with the documented formula.
	ha1 := md5hex("admin:Digest:abc:P@ssw0rd")
	ha2 := md5hex("POST:" + authURI)
	want := md5hex(ha1 + ":nonce123:00000002:cnonceXYZ:auth:" + ha2)
	if got != want {
		t.Fatalf("digest = %s, want %s", got, want)
	}
}
