package radio

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func (s *Service) resolveEnlaceSessions(ctx context.Context, profile string, sessions []equipment.EnlaceSession) Code {
	store, ok := s.Store.(equipment.PPPCatalogStore)
	if !ok {
		return Unavailable
	}
	// ignored: catalogue timestamps and truncation do not identify an interface.
	entries, _, _, code := store.ReadPPPCatalog(ctx, profile, "")
	if code != OK {
		return code
	}
	for i := range sessions {
		session := &sessions[i]
		iface := ResolveEnlaceInterface(session.Row, session.Concentrator, entries)
		if session.Row.Context != nil {
			c := *session.Row.Context
			c.Interface = iface
			session.Row.Context = &c
		}
	}
	return OK
}

func ResolveEnlaceInterface(row PPRow, concentrator string, entries []equipment.PPPEntry) string {
	if row.Context == nil || row.Context.Association != "associated" {
		return ""
	}
	found := ""
	matches := 0
	for _, e := range entries {
		if !enlaceServerMatches(row, concentrator, e) {
			continue
		}
		if found != "" && found != e.Interface {
			return ""
		}
		found = e.Interface
		matches++
	}
	if matches > 0 {
		return found
	}
	return ""
}

func (s *Service) validEnlaceInterface(ctx context.Context, profile string, in RadioInput) Code {
	store, ok := s.Store.(equipment.PPPCatalogStore)
	if !ok {
		return Unavailable
	}
	// ignored: catalogue timestamps and truncation do not identify an interface.
	entries, _, _, code := store.ReadPPPCatalog(ctx, profile, in.DeviceID)
	if code != OK {
		return code
	}
	for _, e := range entries {
		if e.Kind == "server" && e.Interface == in.Interface && (e.Invalid == nil || !*e.Invalid) {
			return OK
		}
	}
	return Invalid
}

func enlaceServerMatches(row PPRow, concentrator string, e equipment.PPPEntry) bool {
	if e.Kind != "server" || e.DeviceID != concentrator || e.Invalid != nil && *e.Invalid {
		return false
	}
	if e.Service == "" || e.Service != row.Context.Service {
		return false
	}
	return row.Context.VLAN == nil || e.VLAN == nil || *row.Context.VLAN == *e.VLAN
}
