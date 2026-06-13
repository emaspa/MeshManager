package amt

import (
	"fmt"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	amtboot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimboot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"

	"github.com/emaspa/meshmanager/amtd/internal/redirect"
)

// StartIDER opens an IDE-R session serving isoPath as a virtual CD-ROM. When
// bootNow is true it also configures the device to boot from IDE-R and resets
// it, so the machine boots the ISO. The IDE-R connection stays open (served by
// the sidecar) until StopIDER, disconnect, or shutdown.
func (s *Session) StartIDER(isoPath string, bootNow bool) error {
	s.iderMu.Lock()
	if s.ider != nil {
		s.ider.Close()
		s.ider = nil
	}
	s.iderMu.Unlock()

	sess, err := redirect.StartIDER(s.RedirectionTarget(), isoPath)
	if err != nil {
		return err
	}
	s.iderMu.Lock()
	s.ider = sess
	s.iderMu.Unlock()

	if bootNow {
		if err := s.configureIDERBoot(); err != nil {
			return fmt.Errorf("IDE-R serving, but boot config failed: %w", err)
		}
		if _, err := s.Power("reset"); err != nil {
			return fmt.Errorf("IDE-R serving, but reset failed: %w", err)
		}
	}
	return nil
}

// configureIDERBoot sets the one-time boot to the IDE-R CD-ROM.
func (s *Session) configureIDERBoot() error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		req := amtboot.BootSettingDataRequest{
			InstanceID:     bootSettingsInstanceID,
			ElementName:    "Intel(r) AMT Boot Configuration Settings",
			UseIDER:        true,
			IDERBootDevice: 1, // 1 = CD/DVD, 0 = floppy
			BIOSSetup:      false,
			BIOSPause:      false,
			BootMediaIndex: 0,
		}
		if _, err := m.AMT.BootSettingData.Put(req); err != nil {
			return fmt.Errorf("set boot settings: %w", err)
		}
		if _, err := m.CIM.BootConfigSetting.ChangeBootOrder(cimboot.CD); err != nil {
			return fmt.Errorf("change boot order: %w", err)
		}
		if _, err := m.CIM.BootService.SetBootConfigRole(bootConfigInstanceID, 1); err != nil {
			return fmt.Errorf("set boot config role: %w", err)
		}
		return nil
	})
}

// StopIDER tears down any active IDE-R session.
func (s *Session) StopIDER() {
	s.iderMu.Lock()
	defer s.iderMu.Unlock()
	if s.ider != nil {
		s.ider.Close()
		s.ider = nil
	}
}

// IDERStatus returns the live IDE-R stats and whether a session is active.
func (s *Session) IDERStatus() (redirect.IDERStats, bool) {
	s.iderMu.Lock()
	defer s.iderMu.Unlock()
	if s.ider == nil {
		return redirect.IDERStats{}, false
	}
	return s.ider.Stats(), true
}
