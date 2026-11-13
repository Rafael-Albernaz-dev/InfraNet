package routerssh

import (
	"context"
	"infradesktop/internal/adapters/routerssh/pppoeparse"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pppoe"
	"infradesktop/internal/modules/pulse"
	"time"
)

func (s *liveSession) readSessionContext(ctx context.Context, o *equipment.PPObservation) {
	o.ContextReadAt = equipment.PPTime(s.contextAt)
	now := time.Now()
	if !pppoe.NeedsSessionContext(o.Rows, s.contextAt, now) {
		return
	}
	// Failed enrichment also consumes this interval; the basic session reading survives.
	s.contextAt = now
	o.ContextReadAt = equipment.PPTime(now)
	started := time.Now()
	text, fault := s.runLive(ctx, pppoeparse.PPContextCommand, equipment.PPWireBytes)
	if fault != pulse.ReadOK {
		return
	}
	tunnels, version, up, code := pppoeparse.ParsePPContext(text)
	age := equipment.PPInterval{Known: true, Max: time.Since(started).Nanoseconds()}
	if code == equipment.OK && version == o.Provenance.Version && up.Known &&
		up.Min > age.Max && up.Max >= o.Provenance.RouterUptime.Min {
		equipment.AssociatePPTunnels(o.Rows, tunnels, age)
	}
}

// SeedSessionContext restores the last attempt's cooldown after SSH reconnects.
func (s *liveSession) SeedSessionContext(previous *equipment.PPObservation) {
	if previous == nil {
		return
	}
	s.previous = previous
	if previous.ContextReadAt != "" {
		// ignored: An invalid optional timestamp leaves the cooldown unset.
		s.contextAt, _ = time.Parse(time.RFC3339Nano, previous.ContextReadAt)
	}
}
