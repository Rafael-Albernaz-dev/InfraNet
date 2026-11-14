package radio

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func (s *Service) enlaceLinks(ctx context.Context, profile string, sessions []equipment.EnlaceSession) (map[string]RadioLink, Code) {
	store, ok := s.Store.(interface {
		PPPRadioLinks(context.Context, string, string) ([]RadioLink, Code)
	})
	if !ok {
		return nil, Unavailable
	}
	out := map[string]RadioLink{}
	read := map[string]bool{}
	for _, session := range sessions {
		if read[session.Concentrator] {
			continue
		}
		read[session.Concentrator] = true
		links, code := store.PPPRadioLinks(ctx, profile, session.Concentrator)
		if code != OK {
			return nil, code
		}
		for _, link := range links {
			out[enlaceKey(profile, link.Concentrator, link.Login)] = link
		}
	}
	return out, OK
}

// FollowEnlaceSessions preserves the pin and invalidates old readings by revision.
func (s *Service) FollowEnlaceSessions(ctx context.Context, profile string, sessions []equipment.EnlaceSession) Code {
	links, code := s.enlaceLinks(ctx, profile, sessions)
	if code != OK {
		return code
	}
	by, ambiguous := uniqueEnlaceSessions(profile, sessions)
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return code
	}
	addresses := map[string]string{}
	for key, link := range links {
		d := device(&state, link.CPE)
		session, exists := by[key]
		if d == nil || d.CPELogin == nil || !exists || ambiguous[key] {
			continue
		}
		if ip, changed := FollowLogin(d.Address, session.Row); changed {
			addresses[d.ID] = ip
		}
	}
	if len(addresses) == 0 {
		return OK
	}
	return s.applyEnlaceAddresses(ctx, profile, state, addresses)
}
func uniqueEnlaceSessions(profile string, sessions []equipment.EnlaceSession) (map[string]equipment.EnlaceSession, map[string]bool) {
	by := map[string]equipment.EnlaceSession{}
	ambiguous := map[string]bool{}
	for _, session := range sessions {
		key := enlaceKey(profile, session.Concentrator, session.Row.Login)
		// ignored: only membership is used.
		if _, exists := by[key]; exists {
			ambiguous[key] = true
		}
		by[key] = session
	}
	return by, ambiguous
}
func (s *Service) applyEnlaceAddresses(ctx context.Context, profile string, state State, addresses map[string]string) Code {
	return s.Store.UpdateEquipment(ctx, profile, func(current *State) Code {
		for _, old := range state.Devices {
			address, exists := addresses[old.ID]
			if !exists {
				continue
			}
			d := device(current, old.ID)
			if d == nil || d.Revision != old.Revision || d.CPELogin == nil || d.Fingerprint != old.Fingerprint {
				return Revision
			}
			d.Address = address
			d.Revision++
		}
		return OK
	})
}
