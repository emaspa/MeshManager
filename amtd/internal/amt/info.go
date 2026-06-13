package amt

import (
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
)

// DeviceInfo is the high-level "who is this machine" summary shown on connect.
type DeviceInfo struct {
	UUID            string            `json:"uuid"`
	Hostname        string            `json:"hostname"`
	DomainName      string            `json:"domainName"`
	DigestRealm     string            `json:"digestRealm"`
	NetworkEnabled  bool              `json:"networkEnabled"`
	Versions        map[string]string `json:"versions"`        // component -> version (AMT, BIOS, ...)
	ProvisioningState string          `json:"provisioningState"`
	ControlMode     string            `json:"controlMode"`
}

// Info gathers identity, general settings and firmware versions for a device.
func (s *Session) Info() (DeviceInfo, error) {
	var info DeviceInfo
	info.Versions = map[string]string{}

	err := s.withWSMAN(func(m *wsman.Messages) error {
		// UUID
		if resp, err := m.AMT.SetupAndConfigurationService.GetUUID(); err == nil {
			if uuid, derr := resp.DecodeUUID(); derr == nil {
				info.UUID = uuid
			}
		}

		// General settings (hostname, domain, realm).
		if resp, err := m.AMT.GeneralSettings.Get(); err == nil {
			g := resp.Body.GetResponse
			info.Hostname = g.HostName
			info.DomainName = g.DomainName
			info.DigestRealm = g.DigestRealm
			info.NetworkEnabled = g.NetworkInterfaceEnabled
		}

		// Provisioning / control mode.
		if resp, err := m.AMT.SetupAndConfigurationService.Get(); err == nil {
			info.ProvisioningState = resp.Body.GetResponse.ProvisioningState.String()
			info.ControlMode = resp.Body.GetResponse.ProvisioningMode.String()
		}

		// Firmware/component versions via CIM_SoftwareIdentity.
		if enum, err := m.CIM.SoftwareIdentity.Enumerate(); err == nil {
			if pull, err := m.CIM.SoftwareIdentity.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, v := range pull.Body.PullResponse.SoftwareIdentityItems {
					id := strings.TrimSpace(v.InstanceID)
					ver := strings.TrimSpace(v.VersionString)
					if id != "" && ver != "" {
						info.Versions[id] = ver
					}
				}
			}
		}
		return nil
	})
	return info, err
}
