package composition

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
)

type identityReadSession struct {
	pulse.ReadSession
	inventory liveInventory
	profile   string
	target    equipment.PPTarget
}

func (s *identityReadSession) Read(ctx context.Context, target pulse.ReadTarget) (pulse.Reading, pulse.ReadFault) {
	reading, fault := s.ReadSession.Read(ctx, target)
	if fault != pulse.ReadOK || reading.Identity == "" {
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
