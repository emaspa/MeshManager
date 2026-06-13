package amt

import (
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
)

// EnableRedirection enables both Serial-over-LAN and IDE-R in the AMT
// redirection service. Many platforms ship with these disabled, in which case
// the redirection channel authenticates but the SOL/IDER session never opens.
// Best-effort: the device's control mode may disallow changing it.
func (s *Session) EnableRedirection() error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		_, err := m.AMT.RedirectionService.RequestStateChange(redirection.EnableIDERAndSOL)
		return err
	})
}
