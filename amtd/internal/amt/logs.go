package amt

import (
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
)

// EventLogEntry is a decoded entry from the AMT firmware event log.
type EventLogEntry struct {
	Time        time.Time `json:"time"`
	Description string    `json:"description"`
	Entity      string    `json:"entity"`
	Severity    string    `json:"severity"`
}

// AuditLogEntry is a decoded entry from the AMT audit log.
type AuditLogEntry struct {
	Time      time.Time `json:"time"`
	App       string    `json:"app"`
	Event     string    `json:"event"`
	Initiator string    `json:"initiator"`
	NetAddress string   `json:"netAddress"`
	Extended  string    `json:"extended"`
}

// EventLog reads and decodes the firmware event log (AMT_MessageLog).
func (s *Session) EventLog() ([]EventLogEntry, error) {
	var out []EventLogEntry
	err := s.withWSMAN(func(m *wsman.Messages) error {
		pos, err := m.AMT.MessageLog.PositionToFirstRecord()
		if err != nil {
			return err
		}
		ident := pos.Body.PositionToFirstRecordResponse.IterationIdentifier

		// Pull in batches until the firmware reports no more records.
		for {
			rec, err := m.AMT.MessageLog.GetRecords(ident, 390)
			if err != nil {
				return err
			}
			for _, e := range rec.Body.GetRecordsResponse.RefinedEventData {
				out = append(out, EventLogEntry{
					Time:        e.TimeStamp,
					Description: e.Description,
					Entity:      e.Entity,
					Severity:    e.EventSeverity,
				})
			}
			if rec.Body.GetRecordsResponse.NoMoreRecords || len(rec.Body.GetRecordsResponse.RefinedEventData) == 0 {
				break
			}
			ident = rec.Body.GetRecordsResponse.IterationIdentifier
		}
		return nil
	})
	return out, err
}

// AuditLog reads and decodes the AMT audit log.
func (s *Session) AuditLog() ([]AuditLogEntry, error) {
	var out []AuditLogEntry
	err := s.withWSMAN(func(m *wsman.Messages) error {
		start := 1
		for {
			resp, err := m.AMT.AuditLog.ReadRecords(start)
			if err != nil {
				return err
			}
			records := resp.Body.DecodedRecordsResponse
			for _, r := range records {
				out = append(out, AuditLogEntry{
					Time:       r.Time,
					App:        r.AuditApp,
					Event:      r.Event,
					Initiator:  r.Initiator,
					NetAddress: r.NetAddress,
					Extended:   r.ExStr,
				})
			}
			returned := resp.Body.ReadRecordsResponse.RecordsReturned
			total := resp.Body.ReadRecordsResponse.TotalRecordCount
			if returned == 0 || start+returned > total {
				break
			}
			start += returned
		}
		return nil
	})
	return out, err
}
