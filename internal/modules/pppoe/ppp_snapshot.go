package pppoe

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func (s *Service) pppSnapshot(ctx context.Context, company string, in PPInput, v *PPView) ([]equipment.PPPEntry, Code) {
	store, ok := s.inv.Store.(equipment.PPPCatalogStore)
	if !ok {
		return nil, Unavailable
	}
	entries, dates, limited, code := store.ReadPPPCatalog(ctx, company, in.DeviceID)
	if code != OK {
		return nil, code
	}
	state, code := s.inv.Store.ReadEquipment(ctx, company)
	if code != OK {
		return nil, code
	}
	names := equipment.DisplayNames(state.Devices)
	devices := pppDevices(state.Devices, in.DeviceID)
	filtered := []equipment.PPPEntry{}
	for _, e := range entries {
		if devices[e.DeviceID].ID != "" {
			e.Concentrator = names[e.DeviceID]
			filtered = append(filtered, e)
		}
	}
	sessions := []equipment.PPPEntry{}
	for _, d := range devices {
		rows, c := s.pppDeviceSessions(ctx, company, d, names[d.ID])
		if c != OK {
			return nil, c
		}
		sessions = append(sessions, rows...)
	}
	EnrichPPPEntries(filtered, sessions)
	if code := s.enrichPPPServers(ctx, company, filtered, sessions, names); code != OK {
		return nil, code
	}
	for i := range filtered {
		applyServerDrop(&filtered[i])
	}
	v.CatalogDates, v.CatalogLimited = dates, limited
	return append(filtered, sessions...), OK
}

func pppDevices(devices []Device, id string) map[string]Device {
	out := map[string]Device{}
	for _, d := range devices {
		if d.Role == equipment.RoleConcentrator && (id == "" || id == d.ID) {
			out[d.ID] = d
		}
	}
	return out
}

func (s *Service) pppDeviceSessions(ctx context.Context, company string, d Device, name string) ([]equipment.PPPEntry, Code) {
	store := s.inv.Store.(PPStore)
	current, code := store.PPCurrent(ctx, company, d.ID)
	if code != OK {
		return nil, code
	}
	if current.Current == nil {
		return nil, OK
	}
	o, code := s.ppLoad(ctx, store, company, d.ID, current.Current.ID)
	if code != OK {
		return nil, code
	}
	entries := []equipment.PPPEntry{}
	for _, row := range o.Rows {
		e := equipment.PPPEntry{ID: row.ID, Kind: "session", Name: row.Login, DeviceID: d.ID, Concentrator: name,
			At: o.Provenance.Ended, ObservationID: o.ID, State: row.State, Session: &row, Client: row.Client}
		if row.Context != nil {
			e.Profile = row.Context.Profile
			e.Service = row.Context.Service
			e.Interface = row.Context.Interface
			e.VLAN = row.Context.VLAN
		}
		entries = append(entries, e)
	}
	return entries, OK
}

func applyServerDrop(e *equipment.PPPEntry) {
	if e.Kind != "server" || e.Peak == nil || *e.Peak <= 0 {
		return
	}
	drop := 100 * float64(*e.Peak-e.Online) / float64(*e.Peak)
	if drop < 0 {
		drop = 0
	}
	e.Drop = &drop
}
