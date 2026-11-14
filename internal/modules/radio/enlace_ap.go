package radio

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) enlaceAP(ctx context.Context, profile, id string, active func() bool, reading func()) APSeen {
	failed := APSeen{AP: id, Failed: true}
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return failed
	}
	d := device(&state, id)
	if !validEnlaceAP(d) {
		return failed
	}
	cr := credential(&state, d.CredentialID)
	if cr == nil {
		return failed
	}
	stamp := fmt.Sprintf("%s|%d|%d", PPConnection(*d), d.Revision, cr.Metadata.Revision)
	if cached, ok := s.cachedEnlaceAP(ctx, profile, id, stamp); ok {
		return cached
	}
	d, code = s.prepareEnlaceAP(ctx, profile, *d, active)
	if code != OK {
		return failed
	}
	reading()
	seen := s.readEnlaceAP(ctx, profile, *d, active)
	s.cacheEnlaceAP(profile, id, stamp, seen, s.Now())
	return seen
}
func validEnlaceAP(d *Device) bool { return d != nil && d.Active && d.Role == RoleAP }
func (s *Service) prepareEnlaceAP(ctx context.Context, profile string, d Device, active func() bool) (*Device, Code) {
	if d.Fingerprint == "" {
		if code := s.BasicContact(ctx, profile, d, active); code != OK {
			return nil, code
		}
	}
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return nil, code
	}
	current := device(&state, d.ID)
	if !validEnlaceAP(current) || current.Fingerprint == "" {
		return nil, TrustRequired
	}
	return current, OK
}
func (s *Service) cachedEnlaceAP(ctx context.Context, profile, id, stamp string) (APSeen, bool) {
	s.enlaceMu.Lock()
	cached, exists := s.enlaceCache[profile+"|"+id]
	s.enlaceMu.Unlock()
	if exists && cached.stamp == stamp && freshEnlaceAP(cached.at, s.Now()) {
		return cached.seen, true
	}
	store, ok := s.Store.(RadioStore)
	if !ok {
		return APSeen{}, false
	}
	// ignored: only the compatible reading is needed, not its saved profile.
	reading, _, code := store.RadioCurrent(ctx, profile, id)
	if code != OK || reading == nil || !reading.Compatible {
		return APSeen{}, false
	}
	at, err := time.Parse(time.RFC3339Nano, reading.At)
	if err != nil || !freshEnlaceAP(at, s.Now()) {
		return APSeen{}, false
	}
	seen := seenRadioAP(id, reading)
	s.cacheEnlaceAP(profile, id, stamp, seen, at)
	return seen, true
}
func freshEnlaceAP(at, now time.Time) bool { return !now.Before(at) && now.Sub(at) < 5*time.Minute }
func seenRadioAP(id string, r *RadioReading) APSeen {
	seen := APSeen{AP: id}
	for _, c := range r.Clients {
		signal := ""
		for _, m := range c.Metrics {
			if m.Name == "signal" || m.Unit == "dBm" {
				signal = fmt.Sprintf("%g %s", m.Value, m.Unit)
				break
			}
		}
		seen.Clients = append(seen.Clients, APClient{c.MAC, signal, c.Interface, c.BSSID})
	}
	return seen
}
func (s *Service) cacheEnlaceAP(profile, id, stamp string, seen APSeen, at time.Time) {
	s.enlaceMu.Lock()
	defer s.enlaceMu.Unlock()
	s.enlaceCache[profile+"|"+id] = enlaceAPCache{seen, stamp, at}
}
