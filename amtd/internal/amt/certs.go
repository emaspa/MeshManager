package amt

import (
	"fmt"
	"regexp"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
)

// Certificate is an X.509 certificate stored on the device
// (AMT_PublicKeyCertificate). Read-only view.
type Certificate struct {
	InstanceID  string `json:"instanceId"`
	Name        string `json:"name"`
	Subject     string `json:"subject"`
	Issuer      string `json:"issuer"`
	TrustedRoot bool   `json:"trustedRoot"`
}

// Certificates lists the device's stored certificates (trusted roots + others).
func (s *Session) Certificates() ([]Certificate, error) {
	var out []Certificate
	err := s.withWSMAN(func(m *wsman.Messages) error {
		enum, err := m.AMT.PublicKeyCertificate.Enumerate()
		if err != nil {
			return err
		}
		pull, err := m.AMT.PublicKeyCertificate.Pull(enum.Body.EnumerateResponse.EnumerationContext)
		if err != nil {
			return err
		}
		for _, c := range pull.Body.PullResponse.PublicKeyCertificateItems {
			out = append(out, Certificate{
				InstanceID:  c.InstanceID,
				Name:        c.ElementName,
				Subject:     c.Subject,
				Issuer:      c.Issuer,
				TrustedRoot: c.TrustedRootCertificate,
			})
		}
		return nil
	})
	return out, err
}

var pemStrip = regexp.MustCompile(`-----[^-]+-----|\s`)

// AddTrustedRootCert adds a trusted root CA certificate. Accepts a PEM block or
// a raw base64 DER blob; PEM armor and whitespace are stripped.
func (s *Session) AddTrustedRootCert(cert string) error {
	blob := pemStrip.ReplaceAllString(cert, "")
	if blob == "" {
		return fmt.Errorf("empty certificate")
	}
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.PublicKeyManagementService.AddTrustedRootCertificate(blob)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("add trusted root failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// DeleteCertificate removes a stored certificate by instance id.
func (s *Session) DeleteCertificate(instanceID string) error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.PublicKeyCertificate.Delete(instanceID)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("delete certificate failed (AMT return value %d)", rv)
		}
		return nil
	})
}
