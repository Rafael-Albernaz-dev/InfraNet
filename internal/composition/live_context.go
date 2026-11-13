package composition

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func (s *identityReadSession) seedSessionContext(ctx context.Context) {
	seed, ok := s.ReadSession.(interface {
		SeedSessionContext(*equipment.PPObservation)
	})
	if !ok {
		return
	}
	store, ok := s.inventory.store.(equipment.PPStore)
	if !ok {
		return
	}
	view, code := store.PPCurrent(ctx, s.profile, s.target.Device.ID)
	if code != equipment.OK || view.Current == nil {
		return
	}
	o := view.Current
	if s.previousObservation != nil && s.previousObservation.ID == o.ID {
		return
	}
	p := o.Provenance
	if p.Connection != equipment.PPConnection(s.target.Device) || p.Revision != s.target.Device.Revision ||
		p.CredentialRevision != s.target.Credential.Metadata.Revision {
		return
	}
	for offset := 0; ; {
		page, code := store.PPPage(ctx, s.profile, equipment.PPInput{DeviceID: o.DeviceID, ObservationID: o.ID, Offset: offset})
		if code != equipment.OK {
			return
		}
		o.Rows = append(o.Rows, page.Rows...)
		if !page.More {
			break
		}
		if page.NextOffset <= offset {
			return
		}
		offset = page.NextOffset
	}
	seed.SeedSessionContext(o)
	s.previousObservation = o
}
