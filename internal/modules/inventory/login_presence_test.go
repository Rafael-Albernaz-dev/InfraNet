package inventory

import (
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"reflect"
	"testing"
	"time"
)

func TestEnlaceCPEPulseFollowsLoginSessionWithoutAProbe(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	since, read, last := "2026-11-14T10:00:00Z", "2026-11-14T11:50:00Z", "2026-11-14T11:49:00Z"
	s := Service{Now: func() time.Time { return now }, Pulse: fixedPulse{pulse.Snapshot{Devices: map[string]pulse.DeviceStatus{
		"cpe": {Address: "192.0.2.1", State: pulse.Online, CheckedAt: now, LastReply: now},
	}}}}
	for _, tc := range []struct {
		name    string
		session *equipment.CPESessionPresence
		want    equipment.PulseStatus
	}{
		{"online", &equipment.CPESessionPresence{State: "ONLINE", Since: since, ReadAt: read},
			equipment.PulseStatus{State: "ONLINE", Since: since, CheckedAt: read, LastReply: read}},
		{"offline", &equipment.CPESessionPresence{State: "OFFLINE", Since: since, ReadAt: read, LastOnline: last},
			equipment.PulseStatus{State: "OFFLINE", Since: since, CheckedAt: read, LastReply: last}},
		{"no session", nil, equipment.PulseStatus{}},
		{"no session in 24h", &equipment.CPESessionPresence{
			State: "ONLINE", ReadAt: "2026-11-13T11:59:00Z", LastOnline: "2026-11-13T11:59:00Z"},
			equipment.PulseStatus{LastReply: "2026-11-13T11:59:00Z"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := Device{ID: "cpe", Address: "192.0.2.1", Active: true, CPELogin: &equipment.CPELogin{Login: "client", Session: tc.session}}
			out := s.withPulse([]Device{d}, "p", &View{})
			if !reflect.DeepEqual(out[0].Pulse, &tc.want) {
				t.Fatalf("session presence: got %#v want %#v", out[0].Pulse, tc.want)
			}
			if d.Pulse != nil {
				t.Fatal("projection changed the stored device")
			}
			filter := tc.want.State
			if filter == "" {
				filter = "unknown"
			}
			if !pulseMatches(out[0], filter) {
				t.Fatal("session state excluded by inventory filter")
			}
		})
	}
}
