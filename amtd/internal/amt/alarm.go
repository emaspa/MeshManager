package amt

import (
	"fmt"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	amtalarm "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/alarmclock"
)

// Alarm is a scheduled wake (AMT_AlarmClockService / IPS_AlarmClockOccurrence).
type Alarm struct {
	InstanceID         string `json:"instanceId"`
	Name               string `json:"name"`
	StartTime          string `json:"startTime"` // RFC3339
	Interval           string `json:"interval"`  // ISO-8601 duration, "" = one-time
	DeleteOnCompletion bool   `json:"deleteOnCompletion"`
}

// Alarms lists the device's scheduled wake-ups.
func (s *Session) Alarms() ([]Alarm, error) {
	var out []Alarm
	err := s.withWSMAN(func(m *wsman.Messages) error {
		enum, err := m.IPS.AlarmClockOccurrence.Enumerate()
		if err != nil {
			return err
		}
		pull, err := m.IPS.AlarmClockOccurrence.Pull(enum.Body.EnumerateResponse.EnumerationContext)
		if err != nil {
			return err
		}
		for _, a := range pull.Body.PullResponse.Items {
			out = append(out, Alarm{
				InstanceID:         a.InstanceID,
				Name:               a.ElementName,
				StartTime:          a.StartTime.Datetime.Format(time.RFC3339),
				Interval:           a.Interval.Interval,
				DeleteOnCompletion: a.DeleteOnCompletion,
			})
		}
		return nil
	})
	return out, err
}

// AddAlarm schedules a wake at startTime (RFC3339). intervalMinutes > 0 makes it
// recurring (e.g. 1440 = daily); 0 is a one-time alarm.
func (s *Session) AddAlarm(name, startTime string, intervalMinutes int, deleteOnCompletion bool) error {
	when, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return fmt.Errorf("invalid start time (want RFC3339): %w", err)
	}
	if name == "" {
		name = "Wake " + when.Format("2006-01-02 15:04")
	}
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.AMT.AlarmClockService.AddAlarm(amtalarm.AlarmClockOccurrence{
			InstanceID:         name,
			ElementName:        name,
			StartTime:          when,
			Interval:           intervalMinutes,
			DeleteOnCompletion: deleteOnCompletion,
		})
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("add alarm failed (AMT return value %d)", rv)
		}
		return nil
	})
}

// DeleteAlarm removes a scheduled wake by its instance id.
func (s *Session) DeleteAlarm(instanceID string) error {
	return s.withWSMAN(func(m *wsman.Messages) error {
		resp, err := m.IPS.AlarmClockOccurrence.Delete(instanceID)
		if err != nil {
			return err
		}
		if rv, ok := returnValue(resp.XMLOutput); ok && rv != 0 {
			return fmt.Errorf("delete alarm failed (AMT return value %d)", rv)
		}
		return nil
	})
}
