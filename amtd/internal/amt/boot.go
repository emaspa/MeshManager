package amt

import (
	"fmt"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	amtboot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimboot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
)

// Standard AMT boot-configuration instance identifiers.
const (
	bootSettingsInstanceID = "Intel(r) AMT: Boot Settings 0"
	bootConfigInstanceID   = "Intel(r) AMT: Boot Configuration 0"
)

// BootDevices are the one-time boot targets the UI offers.
var bootSources = map[string]struct {
	source    cimboot.Source
	biosSetup bool
}{
	"pxe":    {cimboot.PXE, false},
	"cd":     {cimboot.CD, false},
	"hdd":    {cimboot.HardDrive, false},
	"bios":   {"", true}, // clear boot order, force BIOS setup screen
	"normal": {"", false},
}

// Boot configures a one-time boot device and then applies a power action so the
// machine boots into it. The sequence follows the AMT SDK: set BootSettingData,
// ChangeBootOrder, mark the config as next-single-use, then power.
func (s *Session) Boot(device, powerAction string) error {
	cfg, ok := bootSources[device]
	if !ok {
		return fmt.Errorf("unknown boot device %q", device)
	}
	if powerAction == "" {
		powerAction = "reset"
	}

	err := s.withWSMAN(func(m *wsman.Messages) error {
		req := amtboot.BootSettingDataRequest{
			InstanceID:         bootSettingsInstanceID,
			ElementName:        "Intel(r) AMT Boot Configuration Settings",
			BIOSSetup:          cfg.biosSetup,
			BIOSPause:          false,
			BootMediaIndex:     0,
			ConfigurationDataReset: false,
			FirmwareVerbosity:  0,
			IDERBootDevice:     0,
			UseIDER:            false,
			UseSOL:             false,
			UseSafeMode:        false,
			UserPasswordBypass: false,
		}
		if _, err := m.AMT.BootSettingData.Put(req); err != nil {
			return fmt.Errorf("set boot settings: %w", err)
		}
		if _, err := m.CIM.BootConfigSetting.ChangeBootOrder(cfg.source); err != nil {
			return fmt.Errorf("change boot order: %w", err)
		}
		// role 1 = IsNextSingleUse: applies on the next boot only.
		if _, err := m.CIM.BootService.SetBootConfigRole(bootConfigInstanceID, 1); err != nil {
			return fmt.Errorf("set boot config role: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = s.Power(powerAction)
	return err
}
