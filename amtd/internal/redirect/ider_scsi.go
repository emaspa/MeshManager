package redirect

import "encoding/binary"

// CD-ROM ATAPI constant pages (from the AMT SDK / MeshCommander).
var (
	modeSenseCD1A = []byte{0x00, 0x12, 0x01, 0x80, 0, 0, 0, 0, 0x1A, 0x0A, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	modeSenseCD1D = []byte{0x00, 0x12, 0x01, 0x80, 0, 0, 0, 0, 0x1D, 0x0A, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	modeSenseCD2A = []byte{0x00, 0x20, 0x01, 0x80, 0, 0, 0, 0, 0x2A, 0x18, 0, 0, 0, 0, 0x20, 0, 0, 0, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	modeSenseCD3F = []byte{0x00, 0x28, 0x01, 0x80, 0, 0, 0, 0, 0x01, 0x06, 0, 0xff, 0, 0, 0, 0, 0x2A, 0x18, 0, 0, 0, 0, 0x02, 0, 0, 0, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	modeSenseCDErr = []byte{0x00, 0x0E, 0x01, 0x80, 0, 0, 0, 0, 0x01, 0x06, 0, 0xFF, 0, 0, 0, 0}

	cfgProfileList = []byte{0x00, 0x00, 0x03, 0x04, 0x00, 0x08, 0x01, 0x00}
	cfgCore        = []byte{0x00, 0x01, 0x03, 0x04, 0x00, 0x00, 0x00, 0x02}
	cfgMorphing    = []byte{0x00, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00}
	cfgRemovable   = []byte{0x00, 0x03, 0x03, 0x04, 0x29, 0x00, 0x00, 0x02}
	cfgRandom      = []byte{0x00, 0x10, 0x01, 0x08, 0x00, 0x00, 0x08, 0x00, 0x00, 0x01, 0x00, 0x00}
	cfgRead        = []byte{0x00, 0x1E, 0x03, 0x00}
	cfgPowerMgmt   = []byte{0x01, 0x00, 0x03, 0x00}
	cfgTimeout     = []byte{0x01, 0x05, 0x03, 0x00}
)

// handleSCSI dispatches an ATAPI CDB. Floppy (0xA0) always reports no medium.
func (s *IDER) handleSCSI(dev byte, cdb []byte, fr, deviceFlags byte) error {
	if dev != devCDROM {
		return s.commandEndResponse(true, 0x02, dev, 0x3a, 0x00) // no medium
	}
	dma := fr & 1

	switch cdb[0] {
	case 0x00: // TEST_UNIT_READY
		if !s.cdromReady {
			s.cdromReady = true
			return s.commandEndResponse(true, 0x06, dev, 0x28, 0x00) // unit attention / became ready
		}
		return s.commandEndResponse(true, 0x00, dev, 0x00, 0x00)

	case 0x08: // READ_6
		lba := uint32(cdb[1]&0x1f)<<16 | uint32(cdb[2])<<8 | uint32(cdb[3])
		n := int(cdb[4])
		if n == 0 {
			n = 256
		}
		return s.sendDiskData(dev, int64(lba), n, dma)

	case 0x28: // READ_10
		lba := binary.BigEndian.Uint32(cdb[2:6])
		n := int(binary.BigEndian.Uint16(cdb[7:9]))
		return s.sendDiskData(dev, int64(lba), n, dma)

	case 0x0a, 0x2a, 0x2e: // WRITE_* — not supported
		return s.commandEndResponse(true, 0x02, dev, 0x3a, 0x00)

	case 0x1a: // MODE_SENSE_6
		if cdb[2] == 0x3f && cdb[3] == 0x00 {
			return s.dataToHost(dev, true, []byte{0, 0x05, 0x80, 0}, dma == 1)
		}
		return s.commandEndResponse(true, 0x05, dev, 0x24, 0x00)

	case 0x1b: // START_STOP
		return s.commandEndResponse(true, 0x00, dev, 0x00, 0x00)

	case 0x1e: // ALLOW_MEDIUM_REMOVAL
		return s.commandEndResponse(true, 0x00, dev, 0x00, 0x00)

	case 0x23: // READ_FORMAT_CAPACITIES
		body := append(beU32(8), 0x00, 0x00, 0x0b, 0x40, 0x02, 0x00, 0x02, 0x00)
		return s.dataToHost(dev, true, body, dma == 1)

	case 0x25: // READ_CAPACITY
		last := uint32(s.size>>cdSectorLog) - 1
		body := append(beU32(last), 0x00, 0x00, 0x08, 0x00) // 2048-byte blocks
		return s.dataToHost(deviceFlags, true, body, dma == 1)

	case 0x43: // READ_TOC
		msf := cdb[1] & 0x02
		format := cdb[2] & 0x07
		if format == 0 {
			format = cdb[9] >> 6
		}
		if format == 1 {
			return s.dataToHost(dev, true, []byte{0x00, 0x0a, 0x01, 0x01, 0x00, 0x14, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00}, dma == 1)
		}
		if msf != 0 {
			return s.dataToHost(dev, true, []byte{0x00, 0x12, 0x01, 0x01, 0x00, 0x14, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x14, 0xaa, 0x00, 0x00, 0x00, 0x34, 0x13}, dma == 1)
		}
		return s.dataToHost(dev, true, []byte{0x00, 0x12, 0x01, 0x01, 0x00, 0x14, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14, 0xaa, 0x00, 0x00, 0x00, 0x00, 0x00}, dma == 1)

	case 0x46: // GET_CONFIGURATION
		sendAll := cdb[1] != 2
		first := binary.BigEndian.Uint16(cdb[2:4])
		buflen := int(binary.BigEndian.Uint16(cdb[7:9]))
		if buflen == 0 {
			return s.dataToHost(dev, true, append(beU32(0x003c), beU32(0x0008)...), dma == 1)
		}
		r := beU32(0x0008) // current profile header
		add := func(code uint16, page []byte) {
			if first == code || (sendAll && first < code) {
				r = append(r, page...)
			}
		}
		if first == 0 {
			r = append(r, cfgProfileList...)
		}
		add(0x1, cfgCore)
		add(0x2, cfgMorphing)
		add(0x3, cfgRemovable)
		add(0x10, cfgRandom)
		add(0x1E, cfgRead)
		add(0x100, cfgPowerMgmt)
		add(0x105, cfgTimeout)
		r = append(beU32(uint32(len(r))), r...)
		if len(r) > buflen {
			r = r[:buflen]
		}
		return s.dataToHost(dev, true, r, dma == 1)

	case 0x4a: // GET_EVENT_STATUS_NOTIFICATION
		if cdb[1] != 0x01 && cdb[4] != 0x10 {
			return s.commandEndResponse(true, 0x05, dev, 0x26, 0x01)
		}
		return s.dataToHost(dev, true, []byte{0x00, 0x02, 0x80, 0x00}, dma == 1) // medium present

	case 0x4c:
		data := make([]byte, 12)
		data = append(data, 0x87, 0x50, 0x03, 0x00, 0x00, 0x00, 0xb0, 0x51, 0x05, 0x20, 0x00)
		return s.sendCommand(0x51, data, true, false)

	case 0x51: // READ_DISC_INFO
		return s.commandEndResponse(false, 0x05, dev, 0x20, 0x00)

	case 0x5a: // MODE_SENSE_10
		buflen := int(binary.BigEndian.Uint16(cdb[7:9]))
		if buflen == 0 {
			return s.dataToHost(dev, true, append(beU32(0x003c), beU32(0x0008)...), dma == 1)
		}
		var page []byte
		switch cdb[2] & 0x3f {
		case 0x01:
			page = modeSenseCDErr
		case 0x3f:
			page = modeSenseCD3F
		case 0x1A:
			page = modeSenseCD1A
		case 0x1D:
			page = modeSenseCD1D
		case 0x2A:
			page = modeSenseCD2A
		}
		if page == nil {
			return s.commandEndResponse(false, 0x05, dev, 0x20, 0x00)
		}
		return s.dataToHost(dev, true, page, dma == 1)

	default:
		return s.commandEndResponse(false, 0x05, dev, 0x20, 0x00)
	}
}

// sendDiskData serves CD sectors [lba, lba+n) from the ISO, chunked by readbfr.
func (s *IDER) sendDiskData(dev byte, lba int64, n int, dma byte) error {
	mediaBlocks := s.size >> cdSectorLog
	if lba+int64(n) > mediaBlocks {
		return s.commandEndResponse(true, 0x05, dev, 0x21, 0x00) // LBA out of range
	}
	if n == 0 {
		return s.commandEndResponse(true, 0x00, dev, 0x00, 0x00)
	}

	offset := lba << cdSectorLog
	remaining := int64(n) << cdSectorLog
	buf := make([]byte, s.readbfr)
	for remaining > 0 {
		chunk := int64(s.readbfr)
		if chunk > remaining {
			chunk = remaining
		}
		if _, err := s.iso.ReadAt(buf[:chunk], offset); err != nil {
			return s.commandEndResponse(true, 0x03, dev, 0x11, 0x00) // medium read error
		}
		offset += chunk
		remaining -= chunk
		if err := s.dataToHost(dev, remaining == 0, buf[:chunk], dma == 1); err != nil {
			return err
		}
	}
	s.sectorsRead.Add(uint64(n))
	return nil
}

func beU32(v uint32) []byte {
	return binary.BigEndian.AppendUint32(nil, v)
}
