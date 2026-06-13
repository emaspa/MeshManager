package amt

import (
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
)

// NetworkInterface describes one AMT-managed network interface.
type NetworkInterface struct {
	Name           string `json:"name"`
	InstanceID     string `json:"instanceId"`
	MACAddress     string `json:"macAddress"`
	LinkUp         bool   `json:"linkUp"`
	DHCPEnabled    bool   `json:"dhcpEnabled"`
	IPAddress      string `json:"ipAddress"`
	SubnetMask     string `json:"subnetMask"`
	DefaultGateway string `json:"defaultGateway"`
	PrimaryDNS     string `json:"primaryDns"`
	SecondaryDNS   string `json:"secondaryDns"`
	SharedMAC      bool   `json:"sharedMac"`
}

// Network enumerates the device's AMT_EthernetPortSettings interfaces.
func (s *Session) Network() ([]NetworkInterface, error) {
	var out []NetworkInterface
	err := s.withWSMAN(func(m *wsman.Messages) error {
		enum, err := m.AMT.EthernetPortSettings.Enumerate()
		if err != nil {
			return err
		}
		pull, err := m.AMT.EthernetPortSettings.Pull(enum.Body.EnumerateResponse.EnumerationContext)
		if err != nil {
			return err
		}
		for _, p := range pull.Body.PullResponse.EthernetPortItems {
			out = append(out, NetworkInterface{
				Name:           p.ElementName,
				InstanceID:     p.InstanceID,
				MACAddress:     p.MACAddress,
				LinkUp:         p.LinkIsUp,
				DHCPEnabled:    p.DHCPEnabled,
				IPAddress:      p.IPAddress,
				SubnetMask:     p.SubnetMask,
				DefaultGateway: p.DefaultGateway,
				PrimaryDNS:     p.PrimaryDNS,
				SecondaryDNS:   p.SecondaryDNS,
				SharedMAC:      p.SharedMAC,
			})
		}
		return nil
	})
	return out, err
}
