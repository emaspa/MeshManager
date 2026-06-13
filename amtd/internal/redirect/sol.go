// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 MeshManager authors
//
// The SOL protocol here is derived from MeshCommander
// (https://github.com/Ylianst/MeshCommander), Copyright Ylian Saint-Hilaire,
// Apache-2.0. Ported to Go and modified.

package redirect

import (
	"encoding/binary"
	"fmt"
	"time"
)

// SOL is an active Serial-over-LAN session. Bytes from the device arrive on
// Output(); host keystrokes are sent with Write.
type SOL struct {
	conn *Conn
	out  chan []byte
	done chan struct{}
}

// StartSOL connects, authenticates, negotiates SOL settings and opens the
// serial channel. On success a reader goroutine and keepalive ticker run until
// Close (or a connection error) tears the session down.
func StartSOL(t Target) (*SOL, error) {
	c, err := Connect(t, ProtocolSOL)
	if err != nil {
		return nil, err
	}

	// Send SOL channel settings (0x20): conservative buffer/timeout defaults
	// matching the reference console; heartbeat disabled (we keepalive instead).
	settings := []byte{0x20, 0x00, 0x00, 0x00}
	settings = binary.LittleEndian.AppendUint32(settings, c.nextSeq())
	for _, v := range []uint16{10000, 100, 0, 10000, 100, 0} { // MaxTx, TxTo, TxOverflowTo, RxTo, RxFlushTo, Heartbeat
		settings = binary.LittleEndian.AppendUint16(settings, v)
	}
	settings = binary.LittleEndian.AppendUint32(settings, 0)
	if err := c.write(settings); err != nil {
		c.Close()
		return nil, err
	}

	// Wait for the settings reply (0x21), tolerating interleaved frames.
	for {
		cmd, _, err := c.readMessage()
		if err != nil {
			c.Close()
			return nil, err
		}
		if cmd == 0x21 {
			break
		}
		if cmd == 0x29 || cmd == 0x2B || cmd == 0xF0 {
			continue
		}
		c.Close()
		return nil, fmt.Errorf("unexpected command 0x%02x while opening SOL", cmd)
	}

	// Open the SOL data channel (0x27).
	open := []byte{0x27, 0x00, 0x00, 0x00}
	open = binary.LittleEndian.AppendUint32(open, c.nextSeq())
	open = append(open, 0x00, 0x00, 0x1B, 0x00, 0x00, 0x00)
	if err := c.write(open); err != nil {
		c.Close()
		return nil, err
	}
	c.ClearDeadline() // entering continuous streaming

	s := &SOL{conn: c, out: make(chan []byte, 64), done: make(chan struct{})}
	go s.readLoop()
	go s.keepAlive()
	return s, nil
}

// Output is the channel of decoded serial data coming from the device. It is
// closed when the session ends.
func (s *SOL) Output() <-chan []byte { return s.out }

// Write sends host keystrokes/input to the device's serial console.
func (s *SOL) Write(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	msg := []byte{0x28, 0x00, 0x00, 0x00}
	msg = binary.LittleEndian.AppendUint32(msg, s.conn.nextSeq())
	msg = binary.LittleEndian.AppendUint16(msg, uint16(len(p)))
	msg = append(msg, p...)
	return s.conn.write(msg)
}

// Close ends the session and releases the connection.
func (s *SOL) Close() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	s.conn.Close()
}

func (s *SOL) readLoop() {
	defer close(s.out)
	for {
		cmd, data, err := s.conn.readMessage()
		if err != nil {
			return
		}
		switch cmd {
		case 0x2A: // incoming display data; payload begins at offset 10 (data[9:])
			if len(data) > 9 {
				payload := make([]byte, len(data)-9)
				copy(payload, data[9:])
				select {
				case s.out <- payload:
				case <-s.done:
					return
				}
			}
		case 0x29, 0x2B, 0xF0:
			// serial-settings echo / keepalive / recording notice: ignore
		default:
			// Unknown frame: keep reading rather than dropping the session.
		}
	}
}

func (s *SOL) keepAlive() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-t.C:
			msg := []byte{0x2B, 0x00, 0x00, 0x00}
			msg = binary.LittleEndian.AppendUint32(msg, s.conn.nextSeq())
			if err := s.conn.write(msg); err != nil {
				return
			}
		}
	}
}
