package inventory

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"time"
)

// BasicContact keeps the same pin and guarded first-contact adapter on later
// rounds. A refused password is retried only after its credential changes.
func (s *Service) BasicContact(ctx context.Context, profile string, d Device, active func() bool) Code {
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return code
	}
	cr := CredentialByID(&state, d.CredentialID)
	if cr == nil {
		return MissingCredential
	}
	if !BasicContactDue(d, cr.Metadata.Revision, s.Now()) {
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

func BasicContactDue(d Device, credential uint64, now time.Time) bool {
	if !d.Active || d.CredentialID == "" {
		return false
	}
	a := d.LastAttempt
	if a == nil || a.CredentialRevision != credential {
		return true
	}
	if a.Code == equipment.RadioCodeRemoteAuthFailed || a.Code == TrustChanged {
		return false
	}
	if a.DeviceRevision != d.Revision {
		return true
	}
	at, err := time.Parse(time.RFC3339Nano, a.At)
	return err == nil && now.Sub(at) >= 10*time.Minute
}
