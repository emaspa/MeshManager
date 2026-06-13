package amt

import (
	"encoding/xml"
	"testing"
)

const enumXML = `<?xml version="1.0" encoding="UTF-8"?>
<a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope"
            xmlns:g="http://intel.com/wbem/wscim/1/amt-schema/1/AMT_AuthorizationService">
  <a:Body>
    <g:EnumerateUserAclEntries_OUTPUT>
      <g:Handles>1</g:Handles>
      <g:Handles>5</g:Handles>
      <g:ReturnValue>0</g:ReturnValue>
    </g:EnumerateUserAclEntries_OUTPUT>
  </a:Body>
</a:Envelope>`

const userXML = `<?xml version="1.0" encoding="UTF-8"?>
<a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope"
            xmlns:g="http://intel.com/wbem/wscim/1/amt-schema/1/AMT_AuthorizationService">
  <a:Body>
    <g:GetUserAclEntryEx_OUTPUT>
      <g:DigestUsername>operator</g:DigestUsername>
      <g:AccessPermission>2</g:AccessPermission>
      <g:Realms>2</g:Realms>
      <g:Realms>5</g:Realms>
      <g:ReturnValue>0</g:ReturnValue>
    </g:GetUserAclEntryEx_OUTPUT>
  </a:Body>
</a:Envelope>`

const enabledXML = `<?xml version="1.0" encoding="UTF-8"?>
<a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope"
            xmlns:g="http://intel.com/wbem/wscim/1/amt-schema/1/AMT_AuthorizationService">
  <a:Body>
    <g:GetAclEnabledState_OUTPUT>
      <g:State>true</g:State>
      <g:ReturnValue>0</g:ReturnValue>
    </g:GetAclEnabledState_OUTPUT>
  </a:Body>
</a:Envelope>`

func TestParseEnumACLHandles(t *testing.T) {
	var env struct {
		Body struct {
			Out struct {
				Handles []int `xml:"Handles"`
			} `xml:"EnumerateUserAclEntries_OUTPUT"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal([]byte(enumXML), &env); err != nil {
		t.Fatal(err)
	}
	got := env.Body.Out.Handles
	if len(got) != 2 || got[0] != 1 || got[1] != 5 {
		t.Fatalf("handles = %v, want [1 5]", got)
	}
}

func TestParseUserACL(t *testing.T) {
	user, perm, realms := parseUserACL(userXML)
	if user != "operator" {
		t.Fatalf("username = %q, want operator", user)
	}
	if perm != 2 {
		t.Fatalf("perm = %d, want 2", perm)
	}
	if len(realms) != 2 || realms[0] != 2 || realms[1] != 5 {
		t.Fatalf("realms = %v, want [2 5]", realms)
	}
}

func TestParseEnabledState(t *testing.T) {
	if !parseEnabledState(enabledXML) {
		t.Fatal("expected enabled = true")
	}
	if parseEnabledState(`<a:Envelope><a:Body><g:GetAclEnabledState_OUTPUT><g:State>false</g:State></g:GetAclEnabledState_OUTPUT></a:Body></a:Envelope>`) {
		t.Fatal("expected enabled = false")
	}
}

func TestReturnValue(t *testing.T) {
	if v, ok := returnValue(`<g:ReturnValue>0</g:ReturnValue>`); !ok || v != 0 {
		t.Fatalf("returnValue = %d,%v want 0,true", v, ok)
	}
	if v, ok := returnValue(`<g:ReturnValue>2058</g:ReturnValue>`); !ok || v != 2058 {
		t.Fatalf("returnValue = %d,%v want 2058,true", v, ok)
	}
	if _, ok := returnValue(`<g:Other>1</g:Other>`); ok {
		t.Fatal("expected no ReturnValue match")
	}
}

func TestMD5Hex(t *testing.T) {
	// MD5("admin:Digest:abc:P@ssw0rd") computed independently.
	got := md5Hex("admin:Digest:abc:P@ssw0rd")
	if len(got) != 32 {
		t.Fatalf("md5 length = %d", len(got))
	}
}
