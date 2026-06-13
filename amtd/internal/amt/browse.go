package amt

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
)

// browserEntry adapts a typed service's Enumerate/Pull to untyped funcs so a
// single reflection-based routine can browse any AMT/CIM/IPS class.
type browserEntry struct {
	enum func() (any, error)
	pull func(string) (any, error)
}

func wrapEnum[R any](f func() (R, error)) func() (any, error) {
	return func() (any, error) { r, e := f(); return r, e }
}

func wrapPull[R any](f func(string) (R, error)) func(string) (any, error) {
	return func(c string) (any, error) { r, e := f(c); return r, e }
}

// browsers builds the class → enumerate/pull table for a connected device. It's
// constructed per call so the closures capture the live wsman.Messages.
func browsers(m *wsman.Messages) map[string]browserEntry {
	return map[string]browserEntry{
		// AMT
		"AMT_GeneralSettings":                 {wrapEnum(m.AMT.GeneralSettings.Enumerate), wrapPull(m.AMT.GeneralSettings.Pull)},
		"AMT_EthernetPortSettings":            {wrapEnum(m.AMT.EthernetPortSettings.Enumerate), wrapPull(m.AMT.EthernetPortSettings.Pull)},
		"AMT_BootSettingData":                 {wrapEnum(m.AMT.BootSettingData.Enumerate), wrapPull(m.AMT.BootSettingData.Pull)},
		"AMT_BootCapabilities":                {wrapEnum(m.AMT.BootCapabilities.Enumerate), wrapPull(m.AMT.BootCapabilities.Pull)},
		"AMT_RedirectionService":              {wrapEnum(m.AMT.RedirectionService.Enumerate), wrapPull(m.AMT.RedirectionService.Pull)},
		"AMT_PublicKeyCertificate":            {wrapEnum(m.AMT.PublicKeyCertificate.Enumerate), wrapPull(m.AMT.PublicKeyCertificate.Pull)},
		"AMT_PublicPrivateKeyPair":            {wrapEnum(m.AMT.PublicPrivateKeyPair.Enumerate), wrapPull(m.AMT.PublicPrivateKeyPair.Pull)},
		"AMT_EnvironmentDetectionSettingData": {wrapEnum(m.AMT.EnvironmentDetectionSettingData.Enumerate), wrapPull(m.AMT.EnvironmentDetectionSettingData.Pull)},
		"AMT_ManagementPresenceRemoteSAP":     {wrapEnum(m.AMT.ManagementPresenceRemoteSAP.Enumerate), wrapPull(m.AMT.ManagementPresenceRemoteSAP.Pull)},
		"AMT_RemoteAccessPolicyRule":          {wrapEnum(m.AMT.RemoteAccessPolicyRule.Enumerate), wrapPull(m.AMT.RemoteAccessPolicyRule.Pull)},
		"AMT_TLSSettingData":                  {wrapEnum(m.AMT.TLSSettingData.Enumerate), wrapPull(m.AMT.TLSSettingData.Pull)},
		"AMT_TimeSynchronizationService":      {wrapEnum(m.AMT.TimeSynchronizationService.Enumerate), wrapPull(m.AMT.TimeSynchronizationService.Pull)},
		"AMT_WiFiPortConfigurationService":    {wrapEnum(m.AMT.WiFiPortConfigurationService.Enumerate), wrapPull(m.AMT.WiFiPortConfigurationService.Pull)},
		// CIM
		"CIM_SoftwareIdentity":  {wrapEnum(m.CIM.SoftwareIdentity.Enumerate), wrapPull(m.CIM.SoftwareIdentity.Pull)},
		"CIM_BIOSElement":       {wrapEnum(m.CIM.BIOSElement.Enumerate), wrapPull(m.CIM.BIOSElement.Pull)},
		"CIM_Chassis":           {wrapEnum(m.CIM.Chassis.Enumerate), wrapPull(m.CIM.Chassis.Pull)},
		"CIM_Processor":         {wrapEnum(m.CIM.Processor.Enumerate), wrapPull(m.CIM.Processor.Pull)},
		"CIM_PhysicalMemory":    {wrapEnum(m.CIM.PhysicalMemory.Enumerate), wrapPull(m.CIM.PhysicalMemory.Pull)},
		"CIM_MediaAccessDevice": {wrapEnum(m.CIM.MediaAccessDevice.Enumerate), wrapPull(m.CIM.MediaAccessDevice.Pull)},
		"CIM_KVMRedirectionSAP": {wrapEnum(m.CIM.KVMRedirectionSAP.Enumerate), wrapPull(m.CIM.KVMRedirectionSAP.Pull)},
		"CIM_WiFiEndpointSettings": {wrapEnum(m.CIM.WiFiEndpointSettings.Enumerate), wrapPull(m.CIM.WiFiEndpointSettings.Pull)},
		// IPS
		"IPS_OptInService":              {wrapEnum(m.IPS.OptInService.Enumerate), wrapPull(m.IPS.OptInService.Pull)},
		"IPS_KVMRedirectionSettingData": {wrapEnum(m.IPS.KVMRedirectionSettingData.Enumerate), wrapPull(m.IPS.KVMRedirectionSettingData.Pull)},
	}
}

// BrowseClasses returns the sorted list of browsable WS-MAN class names.
func (s *Session) BrowseClasses() []string {
	names := make([]string, 0)
	for k := range browsers(&s.wsman) {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Browse enumerates a WS-MAN class and returns its Pull response body (the
// Items) for display. Read-only.
func (s *Session) Browse(className string) (any, error) {
	var result any
	err := s.withWSMAN(func(m *wsman.Messages) error {
		entry, ok := browsers(m)[className]
		if !ok {
			return fmt.Errorf("unknown or unsupported class %q", className)
		}
		enumResp, err := entry.enum()
		if err != nil {
			return err
		}
		ctx := nestedString(enumResp, "Body", "EnumerateResponse", "EnumerationContext")
		pullResp, err := entry.pull(ctx)
		if err != nil {
			return err
		}
		result = nestedField(pullResp, "Body", "PullResponse")
		return nil
	})
	return result, err
}

// nestedField walks exported struct fields by name (dereferencing pointers).
func nestedField(v any, path ...string) any {
	rv := reflect.ValueOf(v)
	for _, p := range path {
		for rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() != reflect.Struct {
			return nil
		}
		rv = rv.FieldByName(p)
		if !rv.IsValid() {
			return nil
		}
	}
	if !rv.IsValid() || !rv.CanInterface() {
		return nil
	}
	return rv.Interface()
}

func nestedString(v any, path ...string) string {
	f := nestedField(v, path...)
	if s, ok := f.(string); ok {
		return s
	}
	return ""
}
