package amt

import (
	"fmt"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ethernetport"
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

// WiredConfig describes the desired wired-interface IP configuration.
type WiredConfig struct {
	InstanceID     string `json:"instanceId"`
	DHCP           bool   `json:"dhcp"`
	IPAddress      string `json:"ipAddress"`
	SubnetMask     string `json:"subnetMask"`
	DefaultGateway string `json:"defaultGateway"`
	PrimaryDNS     string `json:"primaryDns"`
	SecondaryDNS   string `json:"secondaryDns"`
}

// SetWiredNetwork reconfigures a wired interface for DHCP or a dedicated static
// IP. It reads the current settings and changes only the IP-related fields, to
// avoid disturbing link policy / VLAN / MAC settings.
//
// WARNING: changing the AMT IP can make the device unreachable on its current
// address; the caller (UI) gates this behind a confirmation.
func (s *Session) SetWiredNetwork(c WiredConfig) error {
	if c.InstanceID == "" {
		return fmt.Errorf("interface instanceId is required")
	}
	if !c.DHCP && (c.IPAddress == "" || c.SubnetMask == "") {
		return fmt.Errorf("static configuration requires IP address and subnet mask")
	}
	return s.withWSMAN(func(m *wsman.Messages) error {
		cur, err := m.AMT.EthernetPortSettings.Get(c.InstanceID)
		if err != nil {
			return fmt.Errorf("read current settings: %w", err)
		}
		g := cur.Body.GetAndPutResponse

		req := ethernetport.SettingsRequest{
			ElementName:                  g.ElementName,
			InstanceID:                   g.InstanceID,
			VLANTag:                      g.VLANTag,
			SharedMAC:                    g.SharedMAC,
			LinkPolicy:                   g.LinkPolicy,
			LinkPreference:               g.LinkPreference,
			ConsoleTcpMaxRetransmissions: ethernetport.ConsoleTCPMaxRetransmissions(g.ConsoleTcpMaxRetransmissions),
			PhysicalConnectionType:       g.PhysicalConnectionType,
			PhysicalNicMedium:            g.PhysicalNicMedium,
		}
		if c.DHCP {
			req.DHCPEnabled = true
			req.IpSyncEnabled = true
			req.SharedStaticIp = false
		} else {
			req.DHCPEnabled = false
			req.IpSyncEnabled = false
			req.SharedStaticIp = false
			req.IPAddress = c.IPAddress
			req.SubnetMask = c.SubnetMask
			req.DefaultGateway = c.DefaultGateway
			req.PrimaryDNS = c.PrimaryDNS
			req.SecondaryDNS = c.SecondaryDNS
		}

		resp, err := m.AMT.EthernetPortSettings.Put(c.InstanceID, req)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("set wired network failed (AMT return value %d)", rv)
		}
		return nil
	})
}
