package inventory

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"time"
)

func ContactDue(device Device, credentialRevision uint64, now time.Time) bool {
	if !device.Active || device.Fingerprint != "" || device.CredentialID == "" {
		return false
	}
	attempt := device.LastAttempt
	if attempt == nil || attempt.DeviceRevision != device.Revision || attempt.CredentialRevision != credentialRevision {
		return true
	}
	if attempt.Code == equipment.RadioCodeRemoteAuthFailed {
		return false
	}
	at, err := time.Parse(time.RFC3339Nano, attempt.At)
	return err == nil && now.Sub(at) >= 10*time.Minute
}

// AutomaticContact is a daemon-only capability; its active-company guard is
// supplied by composition, never by the UI or a network IPC caller.
func (s *Service) AutomaticContact(ctx context.Context, profile string, d Device, active func() bool) Code {
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return code
	}
	cr := CredentialByID(&state, d.CredentialID)
	if cr == nil || !s.AutomaticContactDue(profile, d, cr.Metadata.Revision, s.Now()) {
		return Changed
	}
	guard := func() Code {
		if !active() {
			return Changed
		}
		current, code := s.Store.ReadEquipment(ctx, profile)
		if code != OK {
			return code
		}
		return matchingContactTarget(&current, d, cr.Metadata.Revision)
	}
	result, outcome := s.ProbeFirstContact(ctx, d, *cr, state, guard)
	s.Mu.Lock()
	s.ContactAttempts[profile+":"+d.ID] = contactAttempt{revision: d.Revision, credential: cr.Metadata.Revision, outcome: outcome, at: s.Now()}
	s.Mu.Unlock()
	if code := guard(); code != OK {
		return code
	}
	return s.Store.UpdateEquipment(ctx, profile, func(current *State) Code {
		if code := matchingContactTarget(current, d, cr.Metadata.Revision); code != OK {
			return code
		}
		target := DeviceByID(current, d.ID)
		if outcome == OK {
			acceptContact(target, result, profile, 0, s.Now())
		}
		target.LastAttempt = &Attempt{Code: outcome, State: jobStateForCode(outcome), At: PPTime(s.Now()),
			DeviceRevision: d.Revision, CredentialRevision: cr.Metadata.Revision}
		return OK
	})
}

func matchingContactTarget(state *State, d Device, revision uint64) Code {
	current := DeviceByID(state, d.ID)
	cr := CredentialByID(state, d.CredentialID)
	if current == nil || !current.Active || current.Revision != d.Revision ||
		current.CredentialID != d.CredentialID || current.Fingerprint != d.Fingerprint ||
		!sameDeviceEndpoint(*current, d) || cr == nil || cr.Metadata.Revision != revision {
		return Revision
	}
	return OK
}

type contactAttempt struct {
	revision, credential uint64
	outcome              Code
	at                   time.Time
}

func (s *Service) AutomaticContactDue(profile string, d Device, credential uint64, now time.Time) bool {
	if !ContactDue(d, credential, now) {
		return false
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	previous, exists := s.ContactAttempts[profile+":"+d.ID]
	if !exists || previous.revision != d.Revision || previous.credential != credential {
		return true
	}
	if previous.outcome == equipment.RadioCodeRemoteAuthFailed {
		return false
	}
	return now.Sub(previous.at) >= 10*time.Minute
}
