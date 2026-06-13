package redirect

import (
	"bufio"
	"encoding/binary"
	"net"
	"os"
	"testing"
	"time"
)

// newTestIDER builds an IDER whose writes go to a pipe we can read, backed by a
// temp ISO of the given size.
func newTestIDER(t *testing.T, isoSize int64) (*IDER, net.Conn) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.iso")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(isoSize); err != nil {
		t.Fatal(err)
	}
	client, server := net.Pipe()
	s := &IDER{
		conn:    &Conn{net: client, r: bufio.NewReader(client)},
		iso:     f,
		size:    isoSize,
		readbfr: 8192,
		done:    make(chan struct{}),
	}
	return s, server
}

func readFrame(t *testing.T, server net.Conn) []byte {
	t.Helper()
	server.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := server.Read(buf)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	return buf[:n]
}

func TestIDER_ReadCapacityFraming(t *testing.T) {
	const sectors = 1000
	s, server := newTestIDER(t, sectors*2048)
	defer server.Close()
	defer s.iso.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// READ_CAPACITY (0x25) CDB.
		cdb := make([]byte, 12)
		cdb[0] = 0x25
		_ = s.handleSCSI(devCDROM, cdb, 0, devCDROM)
	}()

	frame := readFrame(t, server)
	<-done
	// Header: cmdid 0x54 (DataToHost), then LE32 sequence.
	if frame[0] != 0x54 {
		t.Fatalf("cmdid = 0x%02x, want 0x54", frame[0])
	}
	if seq := binary.LittleEndian.Uint32(frame[4:8]); seq != 0 {
		t.Fatalf("sequence = %d, want 0", seq)
	}
	// The 8-byte SCSI payload is the last 8 bytes: BE last-LBA + block size.
	payload := frame[len(frame)-8:]
	lastLBA := binary.BigEndian.Uint32(payload[0:4])
	if lastLBA != sectors-1 {
		t.Fatalf("last LBA = %d, want %d", lastLBA, sectors-1)
	}
	if payload[6] != 0x08 { // 2048-byte block size high byte (0x0800)
		t.Fatalf("block size bytes = %v, want CD 2048", payload[4:8])
	}
}

func TestIDER_ReadServesSectors(t *testing.T) {
	s, server := newTestIDER(t, 64*2048)
	defer server.Close()
	defer s.iso.Close()

	// Seed a recognizable byte at the start of LBA 2.
	if _, err := s.iso.WriteAt([]byte{0xAB}, 2*2048); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		cdb := make([]byte, 12)
		cdb[0] = 0x28 // READ_10
		binary.BigEndian.PutUint32(cdb[2:6], 2) // LBA 2
		binary.BigEndian.PutUint16(cdb[7:9], 1) // 1 sector
		_ = s.handleSCSI(devCDROM, cdb, 0, devCDROM)
	}()

	frame := readFrame(t, server)
	<-done
	if frame[0] != 0x54 {
		t.Fatalf("cmdid = 0x%02x, want 0x54 (DataToHost)", frame[0])
	}
	// Sector payload is the trailing 2048 bytes; first byte should be our marker.
	data := frame[len(frame)-2048:]
	if data[0] != 0xAB {
		t.Fatalf("served sector byte = 0x%02x, want 0xAB", data[0])
	}
	if s.sectorsRead.Load() != 1 {
		t.Fatalf("sectorsRead = %d, want 1", s.sectorsRead.Load())
	}
}
