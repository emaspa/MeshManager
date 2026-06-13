package amt

import (
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
)

// Hardware is the consolidated hardware inventory for a device.
type Hardware struct {
	System     SystemInfo      `json:"system"`
	BIOS       BIOSInfo        `json:"bios"`
	Processors []ProcessorInfo `json:"processors"`
	Memory     []MemoryInfo    `json:"memory"`
	Disks      []DiskInfo      `json:"disks"`
}

type BIOSInfo struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
}

type SystemInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	SerialNumber string `json:"serialNumber"`
	Version      string `json:"version"`
	ChassisType  int    `json:"chassisType"`
}

type ProcessorInfo struct {
	ID           string `json:"id"`
	Model        string `json:"model"` // brand/version string (from CIM_Chip)
	Manufacturer string `json:"manufacturer"`
	Family       int    `json:"family"`
	MaxClockMHz  int    `json:"maxClockMhz"`
	CurrentClock int    `json:"currentClockMhz"`
	Stepping     string `json:"stepping"`
	Status       string `json:"status"`
}

type MemoryInfo struct {
	BankLabel    string `json:"bankLabel"`
	CapacityMB   int    `json:"capacityMb"`
	SpeedMHz     int    `json:"speedMhz"`
	Type         string `json:"type"`
	FormFactor   string `json:"formFactor"`
	Manufacturer string `json:"manufacturer"`
	PartNumber   string `json:"partNumber"`
	SerialNumber string `json:"serialNumber"`
}

type DiskInfo struct {
	DeviceID     string `json:"deviceId"`
	Model        string `json:"model"`        // from CIM_PhysicalPackage
	SerialNumber string `json:"serialNumber"` // from CIM_PhysicalPackage
	MaxMediaKB   int    `json:"maxMediaKb"`
	ElementName  string `json:"elementName"`
}

// friendlyCPUStatus turns the library's CPUStatus name into MeshCommander-style
// text (e.g. "CPUEnabled" -> "Enabled").
func friendlyCPUStatus(s string) string {
	switch s {
	case "CPUEnabled":
		return "Enabled"
	case "CPUDisabledByUser":
		return "Disabled (user)"
	case "CPUDisabledByBIOS":
		return "Disabled (BIOS)"
	case "CPUIsIdle":
		return "Idle"
	default:
		return s
	}
}

// memoryFormFactor decodes the SMBIOS form-factor code.
var memoryFormFactor = map[int]string{
	1: "Other", 2: "Unknown", 3: "SIMM", 4: "SIP", 5: "Chip", 6: "DIP",
	7: "ZIP", 8: "Proprietary Card", 9: "DIMM", 10: "TSOP", 11: "Row of chips",
	12: "RIMM", 13: "SODIMM", 14: "SRIMM", 15: "FB-DIMM",
}

// Hardware collects CPU, memory, disk and chassis inventory.
func (s *Session) Hardware() (Hardware, error) {
	var hw Hardware
	err := s.withWSMAN(func(m *wsman.Messages) error {
		// BIOS.
		if resp, err := m.CIM.BIOSElement.Get(); err == nil {
			b := resp.Body.BIOSElementGetResponse
			hw.BIOS = BIOSInfo{Vendor: b.Manufacturer, Version: b.Version}
		}

		// Chassis -> system identity.
		if enum, err := m.CIM.Chassis.Enumerate(); err == nil {
			if pull, err := m.CIM.Chassis.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, c := range pull.Body.PullResponse.PackageItems {
					hw.System = SystemInfo{
						Manufacturer: c.Manufacturer,
						Model:        c.Model,
						SerialNumber: c.SerialNumber,
						Version:      c.Version,
						ChassisType:  int(c.ChassisPackageType),
					}
					break
				}
			}
		}

		// CPU brand/version + manufacturer live in CIM_Chip, paired by index
		// with CIM_Processor (same order, per the AMT inventory layout).
		var chips []struct{ manufacturer, version string }
		if enum, err := m.CIM.Chip.Enumerate(); err == nil {
			if pull, err := m.CIM.Chip.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, c := range pull.Body.PullResponse.ChipItems {
					chips = append(chips, struct{ manufacturer, version string }{c.Manufacturer, c.Version})
				}
			}
		}

		// Processors.
		if enum, err := m.CIM.Processor.Enumerate(); err == nil {
			if pull, err := m.CIM.Processor.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for idx, p := range pull.Body.PullResponse.PackageItems {
					pi := ProcessorInfo{
						ID:           p.DeviceID,
						Model:        p.ElementName,
						Family:       int(p.Family),
						MaxClockMHz:  p.MaxClockSpeed,
						CurrentClock: p.CurrentClockSpeed,
						Stepping:     p.Stepping,
						Status:       friendlyCPUStatus(p.CPUStatus.String()),
					}
					if idx < len(chips) {
						pi.Manufacturer = chips[idx].manufacturer
						if chips[idx].version != "" {
							pi.Model = chips[idx].version
						}
					}
					hw.Processors = append(hw.Processors, pi)
				}
			}
		}

		// Physical memory.
		if enum, err := m.CIM.PhysicalMemory.Enumerate(); err == nil {
			if pull, err := m.CIM.PhysicalMemory.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, mem := range pull.Body.PullResponse.MemoryItems {
					hw.Memory = append(hw.Memory, MemoryInfo{
						BankLabel:    mem.BankLabel,
						CapacityMB:   int(mem.Capacity / (1024 * 1024)),
						SpeedMHz:     mem.Speed,
						Type:         mem.MemoryType.String(),
						FormFactor:   memoryFormFactor[mem.FormFactor],
						Manufacturer: mem.Manufacturer,
						PartNumber:   mem.PartNumber,
						SerialNumber: mem.SerialNumber,
					})
				}
			}
		}

		// Storage model + serial come from CIM_PhysicalPackage, paired with
		// CIM_MediaAccessDevice at index+1 (package[0] is the system enclosure).
		// AMT paginates enumerations, so pull every batch: this list (enclosure +
		// disks + battery + ...) can span multiple pulls and a single pull would
		// drop later disks.
		var pkgs []struct{ model, serial string }
		if enum, err := m.CIM.PhysicalPackage.Enumerate(); err == nil {
			ctx := enum.Body.EnumerateResponse.EnumerationContext
			for range 32 { // hard cap to avoid any pathological loop
				pull, err := m.CIM.PhysicalPackage.Pull(ctx)
				if err != nil {
					break
				}
				for _, p := range pull.Body.PullResponse.PhysicalPackage {
					pkgs = append(pkgs, struct{ model, serial string }{p.Model, p.SerialNumber})
				}
				next := pull.Body.PullResponse.EnumerationContext
				if pull.Body.PullResponse.EndOfSequence.Local != "" || next == "" || next == ctx {
					break
				}
				ctx = next
			}
		}

		// Disks / media access devices.
		if enum, err := m.CIM.MediaAccessDevice.Enumerate(); err == nil {
			if pull, err := m.CIM.MediaAccessDevice.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for idx, d := range pull.Body.PullResponse.MediaAccessDevices {
					di := DiskInfo{
						DeviceID:    d.DeviceID,
						ElementName: d.ElementName,
						MaxMediaKB:  int(d.MaxMediaSize),
					}
					if p := idx + 1; p < len(pkgs) {
						di.Model = pkgs[p].model
						di.SerialNumber = pkgs[p].serial
					}
					hw.Disks = append(hw.Disks, di)
				}
			}
		}
		return nil
	})
	return hw, err
}
