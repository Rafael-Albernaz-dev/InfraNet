package pppoe

import "infradesktop/internal/modules/equipment"

type pppIndex struct {
	secrets       map[string][]equipment.PPPEntry
	registrations map[string]int
	online        map[string]int
	vlans         map[string]*int
	sessions      map[string]equipment.PPPEntry
}

func entryKey(device, name string) string { return device + "\x00" + name }
func indexPPP(catalog, sessions []equipment.PPPEntry) pppIndex {
	idx := pppIndex{map[string][]equipment.PPPEntry{}, map[string]int{}, map[string]int{}, map[string]*int{}, map[string]equipment.PPPEntry{}}
	for _, e := range catalog {
		if e.Kind == "secret" {
			key := entryKey(e.DeviceID, e.Name)
			idx.secrets[key] = append(idx.secrets[key], e)
			idx.registrations[entryKey(e.DeviceID, e.Profile)]++
		}
		if e.Kind == "vlan" {
			idx.vlans[entryKey(e.DeviceID, e.Name)] = e.VLAN
		}
	}
	for _, e := range sessions {
		idx.sessions[entryKey(e.DeviceID, e.Name)] = e
		if e.State != "OFFLINE" {
			idx.online[entryKey(e.DeviceID, e.Profile)]++
			idx.online[entryKey(e.DeviceID, e.Interface+"\x00"+e.Service)]++
		}
	}
	return idx
}

// EnrichPPPEntries joins only within one concentrator and preserves ambiguous secret names.
func EnrichPPPEntries(catalog, sessions []equipment.PPPEntry) {
	idx := indexPPP(catalog, sessions)
	for i := range sessions {
		e := &sessions[i]
		items := idx.secrets[entryKey(e.DeviceID, e.Name)]
		if len(items) == 1 {
			e.Client = items[0].Client
			e.Session.Client = e.Client
		}
	}
	for i := range catalog {
		enrichPPPEntry(&catalog[i], idx)
	}
}
func enrichPPPEntry(e *equipment.PPPEntry, idx pppIndex) {
	key := entryKey(e.DeviceID, e.Name)
	switch e.Kind {
	case "secret":
		if s, ok := idx.sessions[key]; ok {
			e.State = s.State
			e.Session = s.Session
			e.ObservationID = s.ObservationID
		}
		if e.Service == "l2tp" {
			e.Kind = "l2tp"
		}
	case "profile":
		e.Registrations = idx.registrations[key]
		e.Online = idx.online[key]
	case "server":
		e.VLAN = idx.vlans[entryKey(e.DeviceID, e.Interface)]
		e.Online = idx.online[entryKey(e.DeviceID, e.Interface+"\x00"+e.Service)]
	}
}
