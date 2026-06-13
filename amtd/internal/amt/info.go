package amt

import (
	"strings"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
)

// DeviceInfo is the high-level "who is this machine" summary shown on connect.
type DeviceInfo struct {
	UUID              string            `json:"uuid"`
	Hostname          string            `json:"hostname"`
	DomainName        string            `json:"domainName"`
	DigestRealm       string            `json:"digestRealm"`
	NetworkEnabled    bool              `json:"networkEnabled"`
	Versions          map[string]string `json:"versions"` // component -> version (AMT, BIOS, ...)
	ProvisioningState string            `json:"provisioningState"`
	ControlMode       string            `json:"controlMode"`
	ActiveFeatures    []string          `json:"activeFeatures"`
	UserConsent       string            `json:"userConsent"`
	DeviceTime        string            `json:"deviceTime"`
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

		// Active features: SOL/IDE-R from the redirection service, KVM from its SAP.
		if resp, err := m.AMT.RedirectionService.Get(); err == nil {
			g := resp.Body.GetAndPutResponse
			if g.ListenerEnabled {
				info.ActiveFeatures = append(info.ActiveFeatures, "Redirection Port")
			}
			switch int(g.EnabledState) {
			case 32770:
				info.ActiveFeatures = append(info.ActiveFeatures, "Serial-over-LAN")
			case 32769:
				info.ActiveFeatures = append(info.ActiveFeatures, "IDE-Redirect")
			case 32771:
				info.ActiveFeatures = append(info.ActiveFeatures, "Serial-over-LAN", "IDE-Redirect")
			}
		}
		if resp, err := m.CIM.KVMRedirectionSAP.Get(); err == nil {
			if int(resp.Body.GetResponse.EnabledState) == 2 { // Enabled
				info.ActiveFeatures = append(info.ActiveFeatures, "KVM")
			}
		}

		// User consent policy.
		if resp, err := m.IPS.OptInService.Get(); err == nil {
			switch resp.Body.GetAndPutResponse.OptInRequired {
			case 0:
				info.UserConsent = "Not required"
			case 1:
				info.UserConsent = "Required for KVM"
			default:
				info.UserConsent = "Required for all sessions"
			}
		}

		// Device clock.
		if resp, err := m.AMT.TimeSynchronizationService.GetLowAccuracyTimeSynch(); err == nil {
			if ta0 := resp.Body.GetLowAccuracyTimeSynchResponse.Ta0; ta0 > 0 {
				info.DeviceTime = time.Unix(ta0, 0).Format("2006-01-02 15:04:05 MST")
			}
		}
		return nil
	})
	return info, err
}
