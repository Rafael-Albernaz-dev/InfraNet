package inventory

import (
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"time"
)

// PulseReader supplies ephemeral reachability without granting collection authority.
type PulseReader interface {
	Snapshot(string, time.Time) (pulse.Snapshot, bool)
}

func pulseTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
func (s *Service) withPulse(devices []Device, profile string, v *View) []Device {
	out := append([]Device(nil), devices...)
	var snapshot pulse.Snapshot
	if s.Pulse != nil {
		if snap, ok := s.Pulse.Snapshot(profile, s.Now()); ok {
			snapshot = snap
			v.PulseNetwork = string(snap.Network)
		}
	}
	for i := range out {
		d := &out[i]
		if d.CPELogin != nil {
			d.Pulse = loginSessionPulse(*d, s.Now())
			continue
		}
		last := ""
		if d.Pulse != nil {
			last = d.Pulse.LastReply
		}
		d.Pulse = &equipment.PulseStatus{LastReply: last}
		status, ok := snapshot.Devices[d.ID]
		if !d.Active || !ok || status.Address != d.Address {
			continue
		}
		reply := pulseTime(status.LastReply)
		if reply == "" {
			reply = last
		}
		d.Pulse = &equipment.PulseStatus{
			State: string(status.State), Since: pulseTime(status.Since),
			LastReply: reply, CheckedAt: pulseTime(status.CheckedAt),
		}
	}
	return out
}

func loginSessionPulse(d Device, now time.Time) *equipment.PulseStatus {
	result := &equipment.PulseStatus{}
	session := d.CPELogin.Session
	if session == nil {
		return result
	}
	result.LastReply = session.LastOnline
	at, err := time.Parse(time.RFC3339Nano, session.ReadAt)
	if !d.Active || err != nil || now.Before(at) || now.Sub(at) >= 24*time.Hour {
		return result
	}
	if session.State != string(pulse.Online) && session.State != string(pulse.Offline) {
		return result
	}
	result.State, result.Since, result.CheckedAt = session.State, session.Since, session.ReadAt
	if session.State == string(pulse.Online) {
		result.LastReply = session.ReadAt
	}
	return result
}
func pulseMatches(d Device, state string) bool {
	if state == "" {
		return true
	}
	current := ""
	if d.Pulse != nil && d.Active {
		current = d.Pulse.State
	}
	if state == "unknown" {
		return current == ""
	}
	return current == state
}

func validPulseFilter(state string) bool {
	return state == "" || state == string(pulse.Online) || state == string(pulse.Offline) || state == "unknown"
}
