package redirect

import (
	"bufio"
	"fmt"
)

// StartKVM connects, authenticates and opens the KVM redirection channel. After
// it returns, the connection is a raw bidirectional RFB pipe: read the device's
// framebuffer stream from Reader() and send RFB client messages with RawWrite.
//
// Unlike SOL, KVM is not framed by the redirection protocol past this point —
// the device speaks Intel AMT's RFB variant directly over the tunnel.
func StartKVM(t Target) (*Conn, error) {
	c, err := Connect(t, ProtocolKVM)
	if err != nil {
		return nil, err
	}

	// Request the KVM data channel.
	if err := c.write([]byte{0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		c.Close()
		return nil, err
	}

	// Wait for the 0x41 start reply (tolerating the recording notice).
	for {
		cmd, _, err := c.readMessage()
		if err != nil {
			c.Close()
			return nil, err
		}
		switch cmd {
		case 0x41:
			// RFB stream begins immediately after; any bytes already pulled
			// into the bufio reader are preserved for the caller.
			return c, nil
		case 0xF0:
			continue
		default:
			c.Close()
			return nil, fmt.Errorf("unexpected command 0x%02x while opening KVM", cmd)
		}
	}
}

// Reader returns the buffered reader carrying the raw RFB stream from the device.
func (c *Conn) Reader() *bufio.Reader { return c.r }

// RawWrite sends raw RFB client→server bytes to the device.
func (c *Conn) RawWrite(b []byte) error { return c.write(b) }
