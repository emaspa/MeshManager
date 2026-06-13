package amt

import (
	"fmt"
	"net"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/remoteaccess"
)

// MpsServer is a configured Management Presence Server (CIRA gateway).
type MpsServer struct {
	Name       string `json:"name"`
	AccessInfo string `json:"accessInfo"`
	Port       int    `json:"port"`
	CommonName string `json:"commonName"`
}

// CiraPolicy is a remote-access policy rule (when CIRA connects).
type CiraPolicy struct {
	Name    string `json:"name"`
	Trigger string `json:"trigger"`
}

// RemoteAccessConfig is the device's CIRA configuration.
type RemoteAccessConfig struct {
	MpsServers           []MpsServer  `json:"mpsServers"`
	Policies             []CiraPolicy `json:"policies"`
	EnvironmentDetection string       `json:"environmentDetection"`
}

// RemoteAccess lists configured MPS servers and CIRA policies.
func (s *Session) RemoteAccess() (RemoteAccessConfig, error) {
	var cfg RemoteAccessConfig
	err := s.withWSMAN(func(m *wsman.Messages) error {
		if enum, err := m.AMT.ManagementPresenceRemoteSAP.Enumerate(); err == nil {
			if pull, err := m.AMT.ManagementPresenceRemoteSAP.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, sap := range pull.Body.PullResponse.ManagementRemoteItems {
					cfg.MpsServers = append(cfg.MpsServers, MpsServer{
						Name:       sap.Name,
						AccessInfo: sap.AccessInfo,
						Port:       sap.Port,
						CommonName: sap.CN,
					})
				}
			}
		}
		if enum, err := m.AMT.RemoteAccessPolicyRule.Enumerate(); err == nil {
			if pull, err := m.AMT.RemoteAccessPolicyRule.Pull(enum.Body.EnumerateResponse.EnumerationContext); err == nil {
				for _, p := range pull.Body.PullResponse.RemotePolicyRuleItems {
					cfg.Policies = append(cfg.Policies, CiraPolicy{
						Name:    p.PolicyRuleName,
						Trigger: p.Trigger.String(),
					})
				}
			}
		}
		cfg.EnvironmentDetection = "Disabled"
		if resp, err := m.AMT.EnvironmentDetectionSettingData.Get(); err == nil {
			if ds := resp.Body.GetAndPutResponse.DetectionStrings; len(ds) > 0 {
				cfg.EnvironmentDetection = strings.Join(ds, ", ")
			}
		}
		return nil
	})
	return cfg, err
}

// AddMpsServer adds a Management Presence Server using username/password auth.
func (s *Session) AddMpsServer(accessInfo string, port int, username, password, commonName string) error {
	if accessInfo == "" || username == "" || password == "" {
		return fmt.Errorf("server address, username and password are required")
	}
	if port == 0 {
		port = 4433
	}
	if commonName == "" {
		commonName = accessInfo
	}
	format := remoteaccess.FQDN
	if net.ParseIP(accessInfo) != nil {
		format = remoteaccess.IPv4Address
	}
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.RemoteAccessService.AddMPS(remoteaccess.AddMpServerRequest{
			AccessInfo: accessInfo,
			InfoFormat: format,
			Port:       port,
			AuthMethod: remoteaccess.UsernamePasswordAuthentication,
			Username:   username,
			Password:   password,
			CommonName: commonName,
		})
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("add MPS failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// DeleteMpsServer removes an MPS server by its name.
func (s *Session) DeleteMpsServer(name string) error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.ManagementPresenceRemoteSAP.Delete(name)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("delete MPS failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// AddCiraPolicy creates a remote-access policy that uses the given MPS. trigger:
// 0 = user-initiated, 1 = alert, 2 = periodic. tunnelLifeSeconds 0 = no limit.
func (s *Session) AddCiraPolicy(mpsName string, trigger, tunnelLifeSeconds int) error {
	if mpsName == "" {
		return fmt.Errorf("an MPS server is required")
	}
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.RemoteAccessService.AddRemoteAccessPolicyRule(remoteaccess.RemoteAccessPolicyRuleRequest{
			Trigger:        remoteaccess.Trigger(trigger),
			TunnelLifeTime: tunnelLifeSeconds,
			ExtendedData:   "",
		}, mpsName)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("add policy failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// DeleteCiraPolicy removes a remote-access policy by its rule name.
func (s *Session) DeleteCiraPolicy(name string) error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.RemoteAccessPolicyRule.Delete(name)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("delete policy failed (AMT return value %d)", rv)
		}
		return nil
	})
}
