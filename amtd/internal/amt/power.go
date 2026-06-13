package amt

import (
	"fmt"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
)

// PowerStatus is the current power state of a device.
type PowerStatus struct {
	State     int    `json:"state"`     // CIM power state value
	StateName string `json:"stateName"` // human-readable name
	On        bool   `json:"on"`        // convenience: is the machine running
}

// PowerActions maps friendly action names to CIM power state requests. These
// are the actions exposed in the UI power menu.
var PowerActions = map[string]power.PowerState{
	"on":               power.PowerOn,               // 2
	"off":              power.PowerOffHard,           // 8
	"off-graceful":     power.PowerOffSoftGraceful,   // 12
	"reset":            power.MasterBusReset,         // 10
	"reset-graceful":   power.MasterBusResetGraceful, // 14
	"cycle":            power.PowerCycleOffHard,      // 5
	"sleep":            power.SleepDeep,              // 4
	"hibernate":        power.Hibernate,              // 7
	"nmi":              power.DiagnosticInterruptNMI, // 11
}

// PowerState reads the current power state via CIM_AssociatedPowerManagementService.
func (s *Session) PowerState() (PowerStatus, error) {
	var ps PowerStatus
	err := s.withWSMAN(func(m *wsman.Messages) error {
		enum, err := m.CIM.AssociatedPowerManagementService.Enumerate()
		if err != nil {
			return err
		}
		pull, err := m.CIM.AssociatedPowerManagementService.Pull(enum.Body.EnumerateResponse.EnumerationContext)
		if err != nil {
			return err
		}
		items := pull.Body.PullResponse.AssociatedPowerManagementServiceItems
		if len(items) == 0 {
			return fmt.Errorf("no power management service returned")
		}
		state := items[0].PowerState
		ps.State = int(state)
		ps.StateName = state.String()
		ps.On = int(state) == 2 // CIM "On"
		return nil
	})
	return ps, err
}

// Power requests a power state change. action must be a key of PowerActions.
func (s *Session) Power(action string) (int, error) {
	target, ok := PowerActions[action]
	if !ok {
		return 0, fmt.Errorf("unknown power action %q", action)
	}
	var ret int
	err := s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.CIM.PowerManagementService.RequestPowerStateChange(target)
		if err != nil {
			return err
		}
		ret = int(resp.Body.RequestPowerStateChangeResponse.ReturnValue)
		if ret != 0 {
			return fmt.Errorf("power request returned %d", ret)
		}
		return nil
	})
	return ret, err
}
