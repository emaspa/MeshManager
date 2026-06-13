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
	Model        string `json:"model"`
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
	DeviceID    string `json:"deviceId"`
	MaxMediaKB  int    `json:"maxMediaKb"`
	ElementName string `json:"elementName"`
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

		// Processors.
		if enum, err := m.CIM.Processor.Enumerate(); err == nil {
			if pull, err := m.CIM.Processor.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, p := range pull.Body.PullResponse.PackageItems {
					hw.Processors = append(hw.Processors, ProcessorInfo{
						ID:           p.DeviceID,
						Model:        p.ElementName,
						Family:       int(p.Family),
						MaxClockMHz:  p.MaxClockSpeed,
						CurrentClock: p.CurrentClockSpeed,
						Stepping:     p.Stepping,
						Status:       p.HealthState.String(),
					})
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

		// Disks / media access devices.
		if enum, err := m.CIM.MediaAccessDevice.Enumerate(); err == nil {
			if pull, err := m.CIM.MediaAccessDevice.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, d := range pull.Body.PullResponse.MediaAccessDevices {
					hw.Disks = append(hw.Disks, DiskInfo{
						DeviceID:    d.DeviceID,
						ElementName: d.ElementName,
						MaxMediaKB:  int(d.MaxMediaSize),
					})
				}
			}
		}
		return nil
	})
	return hw, err
}
