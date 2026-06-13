package amt

import (
	"crypto/md5" //nolint:gosec // AMT digest accounts mandate MD5
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/authorization"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/methods"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"
)

// Account is an AMT user ACL entry (a digest user account).
type Account struct {
	Handle           int    `json:"handle"`
	Username         string `json:"username"`
	AccessPermission int    `json:"accessPermission"` // 0=local, 1=network, 2=any
	Realms           []int  `json:"realms"`
	Enabled          bool   `json:"enabled"`
}

// Accounts enumerates the device's user ACL entries.
//
// go-wsman-messages exposes the enumerate/get/remove/enable methods but does
// not map their outputs to typed fields, so we parse the raw response XML
// (element local-names, namespaces ignored).
func (s *Session) Accounts() ([]Account, error) {
	var out []Account
	err := s.withWSMAN(func(m *wsman.Messages) error {
		handles, err := enumACLHandles(m)
		if err != nil {
			return err
		}
		for _, h := range handles {
			acc := Account{Handle: h, Enabled: true}
			if resp, err := m.AMT.AuthorizationService.GetUserACLEntryEx(h); err == nil {
				acc.Username, acc.AccessPermission, acc.Realms = parseUserACL(resp.XMLOutput)
			}
			if resp, err := m.AMT.AuthorizationService.GetACLEnabledState(h); err == nil {
				acc.Enabled = parseEnabledState(resp.XMLOutput)
			}
			out = append(out, acc)
		}
		return nil
	})
	return out, err
}

func enumACLHandles(m *wsman.Messages) ([]int, error) {
	resp, err := m.AMT.AuthorizationService.EnumerateUserACLEntries(1)
	if err != nil {
		return nil, err
	}
	var env struct {
		Body struct {
			Out struct {
				Handles []int `xml:"Handles"`
			} `xml:"EnumerateUserAclEntries_OUTPUT"`
		} `xml:"Body"`
	}
	_ = xml.Unmarshal([]byte(resp.XMLOutput), &env)
	return env.Body.Out.Handles, nil
}

func parseUserACL(xmlOut string) (username string, perm int, realms []int) {
	// Repeated <Realms>N</Realms> form (MeshCommander / AMT SDK).
	var env struct {
		Body struct {
			Out struct {
				DigestUsername   string `xml:"DigestUsername"`
				Username         string `xml:"Username"`
				AccessPermission int    `xml:"AccessPermission"`
				Realms           []int  `xml:"Realms"`
			} `xml:"GetUserAclEntryEx_OUTPUT"`
		} `xml:"Body"`
	}
	_ = xml.Unmarshal([]byte(xmlOut), &env)
	username = env.Body.Out.DigestUsername
	if username == "" {
		username = env.Body.Out.Username
	}
	realms = env.Body.Out.Realms

	// Fallback: some firmware nests <Realms><RealmValue>N</RealmValue></Realms>.
	// Parsed separately because one struct can't map both "Realms" and
	// "Realms>RealmValue" without an xml error.
	if len(realms) == 0 {
		var nested struct {
			Body struct {
				Out struct {
					RealmValues []int `xml:"Realms>RealmValue"`
				} `xml:"GetUserAclEntryEx_OUTPUT"`
			} `xml:"Body"`
		}
		_ = xml.Unmarshal([]byte(xmlOut), &nested)
		realms = nested.Body.Out.RealmValues
	}
	return username, env.Body.Out.AccessPermission, realms
}

func parseEnabledState(xmlOut string) bool {
	var env struct {
		Body struct {
			Out struct {
				State string `xml:"State"`
			} `xml:"GetAclEnabledState_OUTPUT"`
		} `xml:"Body"`
	}
	_ = xml.Unmarshal([]byte(xmlOut), &env)
	// Default to enabled: AMT reports the built-in admin (and most users) as
	// enabled, and only an explicit false/0 means disabled. Treating a
	// missing/unparsed State as "disabled" wrongly showed enabled accounts as
	// disabled.
	v := strings.ToLower(strings.TrimSpace(env.Body.Out.State))
	return v != "false" && v != "0"
}

// AddAccount creates a digest user. realms are AMT realm numbers (e.g. 2 =
// Redirection); accessPermission is 0=local, 1=network, 2=any.
//
// The body is hand-crafted to match MeshCommander's proven wire format
// (repeated <h:Realms> elements), sent through the library's message creator.
func (s *Session) AddAccount(username, password string, accessPermission int, realms []int) error {
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}
	if len(username) > 16 {
		return fmt.Errorf("username must be 16 characters or fewer")
	}
	return s.withWSMAN(func(m *wsman.Messages) error {
		gen, err := m.AMT.GeneralSettings.Get()
		if err != nil {
			return fmt.Errorf("read digest realm: %w", err)
		}
		realm := gen.Body.GetResponse.DigestRealm
		digest := md5Hex(fmt.Sprintf("%s:%s:%s", username, realm, password))

		svc := m.AMT.AuthorizationService
		creator := svc.Base.WSManMessageCreator

		var body strings.Builder
		fmt.Fprintf(&body, `<Body><h:AddUserAclEntryEx_INPUT xmlns:h="%sAMT_AuthorizationService">`, creator.ResourceURIBase)
		fmt.Fprintf(&body, `<h:DigestUsername>%s</h:DigestUsername>`, xmlEscape(username))
		fmt.Fprintf(&body, `<h:DigestPassword>%s</h:DigestPassword>`, digest)
		fmt.Fprintf(&body, `<h:AccessPermission>%d</h:AccessPermission>`, accessPermission)
		for _, r := range realms {
			fmt.Fprintf(&body, `<h:Realms>%d</h:Realms>`, r)
		}
		body.WriteString(`</h:AddUserAclEntryEx_INPUT></Body>`)

		action := methods.GenerateAction(authorization.AMTAuthorizationService, authorization.AddUserACLEntryEx)
		header := creator.CreateHeader(action, authorization.AMTAuthorizationService, nil, "", "")
		msg := &client.Message{XMLInput: creator.CreateXML(header, body.String())}
		if err := svc.Base.Execute(msg); err != nil {
			return err
		}
		if rv, ok := returnValue(msg.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("add user failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// RemoveAccount deletes a user ACL entry by handle.
func (s *Session) RemoveAccount(handle int) error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.AuthorizationService.RemoveUserACLEntry(handle)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("remove failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// SetAccountEnabled enables or disables a user ACL entry.
func (s *Session) SetAccountEnabled(handle int, enabled bool) error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.AuthorizationService.SetACLEnabledState(handle, enabled)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("set enabled failed (AMT return value %d)", rv)
		}
		return nil
	})
}

var reReturnValue = regexp.MustCompile(`ReturnValue>\s*(-?\d+)\s*<`)

// returnValue extracts the AMT ReturnValue from any method's response body.
func returnValue(xmlOut string) (int, bool) {
	m := reReturnValue.FindStringSubmatch(xmlOut)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s)) //nolint:gosec
	return hex.EncodeToString(sum[:])
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
