package amt

import (
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
)

// EnableKVM turns on the KVM redirection SAP (CIM_KVMRedirectionSAP). Intel AMT
// ships with KVM disabled on many platforms, in which case the redirection
// channel opens but no framebuffer is ever sent (a black screen). This is
// best-effort: the SAP may already be enabled, or the device's control mode may
// disallow changing it.
func (s *Session) EnableKVM() error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		_, err := m.CIM.KVMRedirectionSAP.RequestStateChange(kvm.RedirectionSAPEnable)
		return err
	})
}
