package amt

import (
	"fmt"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
)

// WiFiProfile is a stored wireless profile (CIM_WiFiEndpointSettings).
type WiFiProfile struct {
	InstanceID string `json:"instanceId"`
	Name       string `json:"name"`
	SSID       string `json:"ssid"`
	Auth       string `json:"auth"`
	Priority   int    `json:"priority"`
}

// WiFiProfiles lists the device's stored wireless profiles.
func (s *Session) WiFiProfiles() ([]WiFiProfile, error) {
	var out []WiFiProfile
	err := s.withWSMAN(func(m *wsman.Messages) error {
		enum, err := m.CIM.WiFiEndpointSettings.Enumerate()
		if err != nil {
			return err
		}
		pull, err := m.CIM.WiFiEndpointSettings.Pull(enum.Body.EnumerateResponse.EnumerationContext)
		if err != nil {
			return err
		}
		for _, p := range pull.Body.PullResponse.EndpointSettingsItems {
			out = append(out, WiFiProfile{
				InstanceID: p.InstanceID,
				Name:       p.ElementName,
				SSID:       p.SSID,
				Auth:       p.AuthenticationMethod.String(),
				Priority:   p.Priority,
			})
		}
		return nil
	})
	return out, err
}

// AddWiFiProfile adds a WPA2-PSK (CCMP) wireless profile. Lower priority numbers
// are tried first.
func (s *Session) AddWiFiProfile(ssid, passphrase string, priority int) error {
	if ssid == "" || passphrase == "" {
		return fmt.Errorf("ssid and passphrase are required")
	}
	return s.withWSMAN(func(m *wsman.Messages) error {
		req := wifi.WiFiEndpointSettingsRequest{
			ElementName:          ssid,
			InstanceID:           "Intel(r) AMT:WiFi Endpoint Settings " + ssid,
			AuthenticationMethod: wifi.AuthenticationMethodWPA2PSK,
			EncryptionMethod:     wifi.EncryptionMethodCCMP,
			SSID:                 ssid,
			Priority:             priority,
			PSKPassPhrase:        passphrase,
			BSSType:              wifi.BSSTypeInfrastructure,
		}
		resp, err := m.AMT.WiFiPortConfigurationService.AddWiFiSettings(
			req, models.IEEE8021xSettings{}, "WiFi Endpoint 0", "", "",
		)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("add WiFi profile failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// DeleteWiFiProfile removes a wireless profile by instance id.
func (s *Session) DeleteWiFiProfile(instanceID string) error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.CIM.WiFiEndpointSettings.Delete(instanceID)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("delete WiFi profile failed (AMT return value %d)", rv)
		}
		return nil
	})
}
