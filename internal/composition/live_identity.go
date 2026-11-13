package composition

import (
	"context"
	"github.com/google/uuid"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"infradesktop/internal/modules/pppoe"
	"infradesktop/internal/modules/pulse"
	"time"
)

type identityReadSession struct {
	pulse.ReadSession
	inventory           liveInventory
	profile             string
	target              equipment.PPTarget
	previous            []equipment.PPTunnel
	previousObservation *equipment.PPObservation
	clock               *inventory.Service
}

func (s *identityReadSession) Read(ctx context.Context, target pulse.ReadTarget) (pulse.Reading, pulse.ReadFault) {
	s.seedSessionContext(ctx)
	reading, fault := s.ReadSession.Read(ctx, target)
	if fault != pulse.ReadOK {
		if s.markGap(ctx) != equipment.OK {
			return pulse.Reading{}, fault
		}
		return reading, fault
	}
	s.markMissingSessions(ctx, reading)
	if reading.Identity == "" {
		return reading, fault
	}
	profile, active := s.inventory.access.ActiveProfile(ctx)
	if !active || profile != s.profile {
		return pulse.Reading{}, pulse.ReadLost
	}
	state, code := s.inventory.store.ReadEquipment(ctx, profile)
	if code != equipment.OK {
		return pulse.Reading{}, pulse.ReadLost
	}
	current, code := liveIdentityDevice(&state, s.target)
	if code != equipment.OK {
		return pulse.Reading{}, pulse.ReadLost
	}
	if reading.At.IsZero() {
		reading.At = time.Now()
	}
	s.prepareSessions(&reading)
	if store, ok := s.inventory.store.(pulseReadingStore); ok {
		if code := store.SavePulseReading(ctx, profile, s.target, reading); code != equipment.OK {
			return pulse.Reading{}, pulse.ReadLost
		}
	}
	s.previous = reading.Tunnels
	if reading.Observation != nil {
		s.previousObservation = reading.Observation
	}
	if current.Identity == reading.Identity {
		return reading, fault
	}
	code = s.inventory.store.UpdateEquipment(ctx, profile, func(state *equipment.State) equipment.Code {
		current, code := liveIdentityDevice(state, s.target)
		if code != equipment.OK {
			return code
		}
		current.Identity = reading.Identity
		return equipment.OK
	})
	if code != equipment.OK {
		return pulse.Reading{}, pulse.ReadLost
	}
	return reading, fault
}

// markGap records that the session list was not read in this cycle.
func (s *identityReadSession) markGap(ctx context.Context) equipment.Code {
	if store, ok := s.inventory.store.(pulseReadingStore); ok {
		return store.MarkPulseGap(ctx, s.profile, s.target, time.Now())
	}
	return equipment.OK
}

// markMissingSessions keeps the device reading of a concentrator whose session
// list failed in this cycle and only records the collection gap.
func (s *identityReadSession) markMissingSessions(ctx context.Context, reading pulse.Reading) {
	if s.target.Device.Role == equipment.RoleConcentrator && reading.Observation == nil {
		// ignored: a failed gap mark is retried by the next failed cycle; the device reading still counts.
		_ = s.markGap(ctx)
	}
}

func liveIdentityDevice(state *equipment.State, target equipment.PPTarget) (*equipment.Device, equipment.Code) {
	for i := range state.Devices {
		current := &state.Devices[i]
		if current.ID != target.Device.ID {
			continue
		}
		if current.Revision != target.Device.Revision || !current.Active ||
			equipment.PPConnection(*current) != equipment.PPConnection(target.Device) {
			return nil, equipment.Revision
		}
		for _, credential := range state.Credentials {
			if credential.Metadata.ID == current.CredentialID && credential.Metadata.Revision == target.Credential.Metadata.Revision {
				return current, equipment.OK
			}
		}
		return nil, equipment.Revision
	}
	return nil, equipment.Changed
}

type pulseReadingStore interface {
	SavePulseReading(context.Context, string, equipment.PPTarget, pulse.Reading) equipment.Code
	MarkPulseGap(context.Context, string, equipment.PPTarget, time.Time) equipment.Code
}

func (s *identityReadSession) prepareSessions(reading *pulse.Reading) {
	o := reading.Observation
	if o == nil {
		return
	}
	o.ID, o.DeviceID = uuid.NewString(), s.target.Device.ID
	o.Provenance.Installation, o.Provenance.Company = s.target.Installation, s.profile
	o.Provenance.DeviceID, o.Provenance.Revision = o.DeviceID, s.target.Device.Revision
	o.Provenance.Connection = equipment.PPConnection(s.target.Device)
	o.Provenance.CredentialRevision = s.target.Credential.Metadata.Revision
	o.Provenance.Started, o.Provenance.Ended = equipment.PPTime(reading.Started), equipment.PPTime(reading.At)
	sample := equipment.PPInterval{}
	if s.clock != nil {
		o.Provenance.Epoch = s.clock.Epoch
		sample = equipment.PPInterval{Known: true,
			Min: reading.Started.Sub(s.clock.Origin).Nanoseconds(),
			Max: reading.At.Sub(s.clock.Origin).Nanoseconds()}
	}
	o.Provenance.Automatic = true
	o.Provenance.Sample = sample
	by := map[string]equipment.PPRow{}
	for i := range o.Rows {
		r := &o.Rows[i]
		r.ID, r.Sample = uuid.NewSHA1(uuid.NameSpaceOID, []byte(o.DeviceID+"\x00"+pppoe.SessionKey(*r))).String(), sample
		by["<pppoe-"+r.Login+">"] = *r
	}
	for i := range reading.Tunnels {
		tunnel := &reading.Tunnels[i]
		r := by[tunnel.Name]
		tunnel.Login, tunnel.Caller, tunnel.Address = r.Login, r.Caller, r.Address
		tunnel.Uptime, tunnel.RouterUptime, tunnel.Sample = r.Uptime, o.Provenance.RouterUptime, sample
	}
	pppoe.CarryObservationContext(s.previousObservation, o)
	pppoe.AverageSessions(o.Rows, s.previous, reading.Tunnels)
}
