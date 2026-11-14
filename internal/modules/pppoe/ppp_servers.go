package pppoe

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"sort"
)

func (s *Service) enrichPPPServers(
	ctx context.Context, company string, catalog, sessions []equipment.PPPEntry, names map[string]string,
) Code {
	store, ok := s.inv.Store.(interface {
		PPContextCounts(context.Context, string, PPInput) (PPView, Code)
	})
	if !ok {
		return Unavailable
	}
	groups := map[string][]equipment.PPGroup{}
	observations := map[string]string{}
	for _, e := range sessions {
		observations[e.DeviceID] = e.ObservationID
	}
	for device, observation := range observations {
		in := PPInput{DeviceID: device, ObservationID: observation}
		for {
			page, code := store.PPContextCounts(ctx, company, in)
			if code != OK {
				return code
			}
			groups[device] = append(groups[device], page.Groups...)
			if !page.More {
				break
			}
			in.Offset = page.NextOffset
		}
	}
	bindings := []equipment.InterfaceAPs{}
	if store, ok := s.inv.Store.(equipment.EnlaceStore); ok {
		var code Code
		bindings, code = store.InterfaceAPs(ctx, company)
		if code != OK {
			return code
		}
	}
	radio, hasRadio := s.inv.Store.(equipment.PPPCatalogStore)
	for i := range catalog {
		e := &catalog[i]
		if e.Kind != "server" {
			continue
		}
		aps := map[string]bool{}
		if hasRadio {
			links, code := radio.PPPRadioLinks(ctx, company, e.DeviceID)
			if code != OK {
				return code
			}
			enrichSessionLinks(sessions, links, names)
			aps = serverAPs(*e, links, sessions, names)
		}
		aps = configuredServerAPs(e, bindings, names, aps)
		for ap := range aps {
			e.APs = append(e.APs, ap)
		}
		sort.Strings(e.APs)
		serverPeak(e, groups[e.DeviceID])
	}
	return OK
}

func serverAPs(e equipment.PPPEntry, links []equipment.RadioLink, sessions []equipment.PPPEntry, names map[string]string) map[string]bool {
	aps := map[string]bool{}
	for _, link := range links {
		if names[link.AP] == "" {
			continue
		}
		for _, s := range sessions {
			if s.DeviceID == e.DeviceID && s.Name == link.Login && s.Interface == e.Interface {
				aps[names[link.AP]] = true
			}
		}
	}
	return aps
}

func serverPeak(e *equipment.PPPEntry, groups []equipment.PPGroup) {
	for _, g := range groups {
		if g.Service == e.Service && g.VLAN != nil && e.VLAN != nil && *g.VLAN == *e.VLAN {
			e.Peak = g.Peak
		}
	}
}

func enrichSessionLinks(sessions []equipment.PPPEntry, links []equipment.RadioLink, names map[string]string) {
	for i := range sessions {
		e := &sessions[i]
		for _, link := range links {
			if e.DeviceID != link.Concentrator || e.Name != link.Login || e.Session == nil {
				continue
			}
			if names[link.AP] != "" {
				e.APs = []string{names[link.AP]}
			}
			e.Session.LinkedDeviceName = names[link.CPE]
		}
	}
}

func configuredServerAPs(e *equipment.PPPEntry, bindings []equipment.InterfaceAPs,
	names map[string]string, aps map[string]bool,
) map[string]bool {
	for _, binding := range bindings {
		if binding.Concentrator == e.DeviceID && binding.Interface == e.Interface {
			aps = map[string]bool{}
			e.APDevices = map[string]string{}
			for _, id := range binding.APs {
				if names[id] != "" {
					aps[names[id]] = true
					e.APDevices[id] = names[id]
				}
			}
		}
	}
	return aps
}
