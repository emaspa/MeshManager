package redirect

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

// IDE-R (IDE Redirection) lets a device boot from a remote CD/ISO image. After
// the redirection handshake (protocol 3), this engine emulates an ATAPI CD-ROM:
// it answers the device's SCSI commands and serves 2048-byte sectors from the
// ISO file. Ported from MeshCommander's IDER module.
//
// Scope: CD-ROM only (the boot-from-ISO case); floppy emulation is omitted.
//
// Wire note: the IDER protocol layer is little-endian; SCSI CDBs and SCSI data
// payloads are big-endian.

const (
	devCDROM    = 0xB0
	cdSectorLog = 11 // 2048-byte sectors
)

// IDERStats reports live transfer progress for the UI.
type IDERStats struct {
	Connected   bool   `json:"connected"`
	BytesToAMT  uint64 `json:"bytesToAmt"`
	SectorsRead uint64 `json:"sectorsRead"`
	ISOSize     int64  `json:"isoSize"`
	Error       string `json:"error"`
}

// IDER is an active IDE-R session serving an ISO to a device.
type IDER struct {
	conn *Conn
	iso  *os.File
	size int64

	outSeq uint32
	inSeq  uint32
	readbfr int

	cdromReady bool
	wmu        sync.Mutex // serialize frame writes

	bytesToAMT  atomic.Uint64
	sectorsRead atomic.Uint64
	connected   atomic.Bool
	errMsg      atomic.Pointer[string]
	done        chan struct{}
}

// StartIDER connects, authenticates, opens the ISO and runs the IDER engine in
// the background until Close or a connection error.
func StartIDER(t Target, isoPath string) (*IDER, error) {
	iso, err := os.Open(isoPath)
	if err != nil {
		return nil, fmt.Errorf("open ISO: %w", err)
	}
	info, err := iso.Stat()
	if err != nil {
		iso.Close()
		return nil, fmt.Errorf("stat ISO: %w", err)
	}

	c, err := Connect(t, ProtocolIDER)
	if err != nil {
		iso.Close()
		return nil, err
	}
	c.ClearDeadline() // IDE-R streams continuously; engine has its own keepalive

	s := &IDER{
		conn:    c,
		iso:     iso,
		size:    info.Size(),
		readbfr: 8192,
		done:    make(chan struct{}),
	}

	// Kick off the session: OPEN_SESSION (0x40).
	open := make([]byte, 0, 10)
	open = binary.LittleEndian.AppendUint16(open, 30000) // rx timeout
	open = binary.LittleEndian.AppendUint16(open, 0)     // tx timeout
	open = binary.LittleEndian.AppendUint16(open, 20000) // heartbeat
	open = binary.LittleEndian.AppendUint32(open, 1)     // version
	if err := s.sendCommand(0x40, open, false, false); err != nil {
		s.cleanup()
		return nil, err
	}
	s.connected.Store(true)

	go s.readLoop()
	return s, nil
}

// Stats returns a snapshot of transfer progress.
func (s *IDER) Stats() IDERStats {
	st := IDERStats{
		Connected:   s.connected.Load(),
		BytesToAMT:  s.bytesToAMT.Load(),
		SectorsRead: s.sectorsRead.Load(),
		ISOSize:     s.size,
	}
	if e := s.errMsg.Load(); e != nil {
		st.Error = *e
	}
	return st
}

// Close terminates the session.
func (s *IDER) Close() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	s.cleanup()
}

func (s *IDER) cleanup() {
	s.connected.Store(false)
	s.conn.Close()
	s.iso.Close()
}

func (s *IDER) fail(err error) {
	msg := err.Error()
	s.errMsg.Store(&msg)
	s.Close()
}

// --- frame I/O ---

func (s *IDER) sendCommand(cmdID byte, data []byte, completed, dma bool) error {
	attributes := byte(0)
	if cmdID > 50 && completed {
		attributes = 2
	}
	if dma {
		attributes++
	}
	frame := []byte{cmdID, 0, 0, attributes}
	frame = binary.LittleEndian.AppendUint32(frame, s.outSeq)
	s.outSeq++
	frame = append(frame, data...)

	s.wmu.Lock()
	err := s.conn.RawWrite(frame)
	s.wmu.Unlock()
	if err == nil {
		s.bytesToAMT.Add(uint64(len(frame)))
	}
	return err
}

func (s *IDER) readLoop() {
	r := s.conn.Reader()
	var acc []byte
	buf := make([]byte, 16384)
	for {
		select {
		case <-s.done:
			return
		default:
		}
		n, err := r.Read(buf)
		if n > 0 {
			acc = append(acc, buf[:n]...)
			for {
				consumed, err := s.processOne(acc)
				if err != nil {
					s.fail(err)
					return
				}
				if consumed == 0 {
					break // need more bytes
				}
				// Verify sequence ordering (header bytes 4..8, little-endian).
				if got := binary.LittleEndian.Uint32(acc[4:8]); got != s.inSeq {
					s.fail(fmt.Errorf("IDER out of sequence: got %d want %d", got, s.inSeq))
					return
				}
				s.inSeq++
				acc = acc[consumed:]
			}
		}
		if err != nil {
			if err != io.EOF {
				s.fail(err)
			} else {
				s.Close()
			}
			return
		}
	}
}

// processOne parses and handles a single IDER frame. Returns bytes consumed, or
// 0 if more data is required.
func (s *IDER) processOne(acc []byte) (int, error) {
	if len(acc) < 8 {
		return 0, nil
	}
	switch acc[0] {
	case 0x41: // OPEN_SESSION reply
		if len(acc) < 30 {
			return 0, nil
		}
		extra := int(acc[29])
		if len(acc) < 30+extra {
			return 0, nil
		}
		readbfr := int(binary.LittleEndian.Uint16(acc[16:18]))
		if readbfr > 0 && readbfr <= 8192 {
			s.readbfr = readbfr
		}
		// Enable IDER on the next reboot: REGS_TOGGLE (3), value 0x01|0x08.
		if err := s.sendDisableEnableFeatures(3, 0x01|0x08); err != nil {
			return 0, err
		}
		return 30 + extra, nil
	case 0x43: // CLOSE
		s.Close()
		return 8, nil
	case 0x44: // PING
		return 8, s.sendCommand(0x45, nil, false, false)
	case 0x45: // PONG
		return 8, nil
	case 0x46: // RESET OCCURRED
		if len(acc) < 9 {
			return 0, nil
		}
		return 9, s.sendCommand(0x47, nil, false, false)
	case 0x49: // STATUS_DATA (DisableEnableFeatures reply)
		if len(acc) < 13 {
			return 0, nil
		}
		typ := acc[8]
		value := binary.LittleEndian.Uint32(acc[9:13])
		if typ == 1 && value&1 != 0 {
			if err := s.sendDisableEnableFeatures(3, 0x01|0x08); err != nil {
				return 0, err
			}
		}
		return 13, nil
	case 0x4A: // ERROR OCCURRED
		if len(acc) < 11 {
			return 0, nil
		}
		return 11, nil
	case 0x4B: // HEARTBEAT
		return 8, nil
	case 0x50: // COMMAND WRITTEN (SCSI command block)
		if len(acc) < 28 {
			return 0, nil
		}
		deviceFlags := acc[14]
		device := byte(devCDROM)
		if deviceFlags&0x10 == 0 {
			device = 0xA0 // floppy - unsupported, will report no media
		}
		cdb := acc[16:28]
		featureRegister := acc[9]
		if err := s.handleSCSI(device, cdb, featureRegister, deviceFlags); err != nil {
			return 0, err
		}
		return 28, nil
	case 0x53: // DATA FROM HOST (write - unsupported)
		if len(acc) < 14 {
			return 0, nil
		}
		dlen := int(binary.LittleEndian.Uint16(acc[9:11]))
		if len(acc) < 14+dlen {
			return 0, nil
		}
		// Report write-protected / no medium.
		resp := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x87, 0x70, 0x03, 0, 0, 0, 0xa0, 0x51, 0x07, 0x27, 0}
		return 14 + dlen, s.sendCommand(0x51, resp, true, false)
	default:
		return 0, fmt.Errorf("unknown IDER command 0x%02x", acc[0])
	}
}

func (s *IDER) sendDisableEnableFeatures(typ byte, value uint32) error {
	data := []byte{typ}
	data = binary.LittleEndian.AppendUint32(data, value)
	return s.sendCommand(0x48, data, false, false)
}

// commandEndResponse sends a SCSI completion / sense (0x51). The branch matches
// MeshCommander's SendCommandEndResponse(e, sense, device, asc, asq).
func (s *IDER) commandEndResponse(e bool, sense, device, asc, asq byte) error {
	var data []byte
	if e {
		data = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xc5, 0, 3, 0, 0, 0, device, 0x50, 0, 0, 0}
	} else {
		data = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x87, sense << 4, 3, 0, 0, 0, device, 0x51, sense, asc, asq}
	}
	return s.sendCommand(0x51, data, true, false)
}

// dataToHost sends SCSI read data (0x54) back to the device.
func (s *IDER) dataToHost(device byte, completed bool, data []byte, dma bool) error {
	dmalen := len(data)
	if dma {
		dmalen = 0
	}
	dl := len(data)
	var hdr []byte
	rw := byte(0xb5)
	if dma {
		rw = 0xb4
	}
	if completed {
		hdr = []byte{0, byte(dl), byte(dl >> 8), 0, rw, 0, 2, 0, byte(dmalen), byte(dmalen >> 8), device, 0x58, 0x85, 0, 3, 0, 0, 0, device, 0x50, 0, 0, 0, 0, 0, 0}
	} else {
		hdr = []byte{0, byte(dl), byte(dl >> 8), 0, rw, 0, 2, 0, byte(dmalen), byte(dmalen >> 8), device, 0x58, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	}
	return s.sendCommand(0x54, append(hdr, data...), completed, dma)
}
