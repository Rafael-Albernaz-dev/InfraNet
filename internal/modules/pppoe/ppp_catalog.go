package pppoe

import (
	"context"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func ValidPPPQuery(q *equipment.PPPQuery) bool {
	if q == nil {
		return false
	}
	switch q.Screen {
	case "sessions", "servers", "secrets", "profiles", "l2tp":
	default:
		return false
	}
	if !utf8.ValidString(q.Search) || utf8.RuneCountInString(q.Search) > 160 || len(q.Filters) > 8 {
		return false
	}
	if q.PageSize != 0 && q.PageSize != 25 && q.PageSize != 50 && q.PageSize != 100 && q.PageSize != 200 {
		return false
	}
	if q.Sort != "" && !PPPColumn(q.Sort) {
		return false
	}
	return validPPPFilters(q.Filters)
}

func PPPColumn(key string) bool {
	switch key {
	case "name", "client", "concentrator", "service", "profile", "state", "interface", "vlan", "mtu", "mru", "authentication",
		"rate", "local_address", "remote_address", "only_one", "caller_id", "last_logged_out",
		"last_caller_id", "last_disconnect_reason",
		"registration", "peak", "drop", "aps", "absent_end", "online", "registrations", "address", "download", "upload",
		"uptime", "session_id", "duplicates", "linked_device":
		return true
	}
	return false
}

func (s *Service) handlePPPList(ctx context.Context, q Command, in PPInput, v *PPView) Code {
	if in.DeviceID != "" && !access.ValidCompanyID(in.DeviceID) {
		return Invalid
	}
	if !ValidPPPQuery(in.PPPQuery) {
		return Invalid
	}
	entries, code := s.pppSnapshot(ctx, q.Scope.Profile, in, v)
	if code != OK {
		return code
	}
	v.Options = PPPOptions(entries, in.PPPQuery.Screen)
	v.Counts = PPPListCounts(entries, in.PPPQuery.Screen)
	entries = filterPPP(entries, in.PPPQuery)
	v.Total = len(entries)
	if in.Offset > v.Total {
		return CursorInvalid
	}
	size := in.PPPQuery.PageSize
	if size == 0 {
		size = 50
	}
	end := min(in.Offset+size, v.Total)
	v.Entries = entries[in.Offset:end]
	v.NextOffset = end
	v.More = end < v.Total
	return OK
}

func filterPPP(entries []equipment.PPPEntry, q *equipment.PPPQuery) []equipment.PPPEntry {
	kinds := map[string]string{"sessions": "session", "servers": "server", "secrets": "secret", "profiles": "profile", "l2tp": "l2tp"}
	out := []equipment.PPPEntry{}
	for _, e := range entries {
		if e.Kind != kinds[q.Screen] {
			continue
		}
		if matchesPPP(e, q) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return lessPPP(out[i], out[j], q) })
	return out
}

func (s *Service) handlePPPSummary(ctx context.Context, q Command, in PPInput, v *PPView) Code {
	if !access.ValidCompanyID(in.DeviceID) {
		return Invalid
	}
	entries, code := s.pppSnapshot(ctx, q.Scope.Profile, in, v)
	if code != OK {
		return code
	}
	summary := equipment.PPPSummary{}
	plans := map[string]bool{}
	for _, e := range entries {
		if e.Kind == "session" {
			if e.State == "OFFLINE" {
				summary.Offline++
			} else {
				summary.Online++
				if e.Profile != "" {
					plans[e.Profile] = true
				}
			}
		}
		if e.Kind == "server" {
			summary.Servers++
			if len(e.APs) == 0 {
				summary.UnlinkedServers++
			}
			if e.Drop != nil && *e.Drop >= 50 && e.Peak != nil && *e.Peak >= 5 {
				summary.FallingServers++
			}
			if PPPValues(e)["state"] == "Inválido" {
				summary.InvalidServers++
			}
		}
	}
	summary.Plans = len(plans)
	now := s.inv.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	summary.DropsToday, code = s.inv.Store.(equipment.PPPCatalogStore).PPPDropsToday(ctx, q.Scope.Profile, in.DeviceID, PPTime(start))
	v.Summary = &summary
	return code
}

func PPPOptions(entries []equipment.PPPEntry, screen string) map[string][]string {
	kinds := map[string]string{"sessions": "session", "servers": "server", "secrets": "secret", "profiles": "profile", "l2tp": "l2tp"}
	sets := map[string]map[string]bool{}
	for _, e := range entries {
		if e.Kind != kinds[screen] {
			continue
		}
		values := PPPValues(e)
		for key, value := range values {
			value = pppFilterValue(e, key, values)
			if value == "" || !pppFilterColumn(screen, key) {
				continue
			}
			if sets[key] == nil {
				sets[key] = map[string]bool{}
			}
			sets[key][value] = true
		}
	}
	out := map[string][]string{}
	for key, values := range sets {
		for value := range values {
			out[key] = append(out[key], value)
		}
		sort.Strings(out[key])
	}
	return out
}

func pppNumericColumn(column string) bool {
	switch column {
	case "online", "registrations", "vlan", "mtu", "mru", "download", "upload", "peak", "drop":
		return true
	}
	return false
}

func validPPPFilters(filters map[string]string) bool {
	for key, value := range filters {
		if !PPPColumn(key) || len(value) > 256 {
			return false
		}
	}
	return true
}
func matchesPPP(e equipment.PPPEntry, q *equipment.PPPQuery) bool {
	values := PPPValues(e)
	for key, want := range q.Filters {
		if want != "" && pppFilterValue(e, key, values) != want {
			return false
		}
	}
	all := []string{}
	for _, value := range values {
		all = append(all, value)
	}
	return strings.Contains(strings.ToLower(strings.Join(all, "\x00")), strings.ToLower(q.Search))
}
func lessPPP(x, y equipment.PPPEntry, q *equipment.PPPQuery) bool {
	a, b := PPPValues(x)[q.Sort], PPPValues(y)[q.Sort]
	if q.Sort == "" {
		a, b = x.Concentrator+x.Name, y.Concentrator+y.Name
	}
	if a == "" || b == "" {
		return a != "" && b == ""
	}
	if pppNumericColumn(q.Sort) {
		return lessPPPNumeric(a, b, x.DeviceID+x.ID, y.DeviceID+y.ID, q.Desc)
	}
	if a == b {
		return x.DeviceID+x.ID < y.DeviceID+y.ID
	}
	if q.Desc {
		return a > b
	}
	return a < b
}
func lessPPPNumeric(a, b, xid, yid string, desc bool) bool {
	x, xe := strconv.ParseFloat(a, 64)
	y, ye := strconv.ParseFloat(b, 64)
	if xe != nil || ye != nil || math.IsNaN(x) || math.IsNaN(y) || x == y {
		return xid < yid
	}
	if desc {
		return x > y
	}
	return x < y
}

func pppFilterColumn(screen, key string) bool {
	if key == "concentrator" {
		return true
	}
	switch screen {
	case "sessions":
		return key == "state" || key == "profile" || key == "vlan" || key == "service" || key == "duplicates"
	case "servers":
		return key == "state" || key == "aps"
	case "secrets":
		return key == "profile" || key == "registration" || key == "state" || key == "last_disconnect_reason"
	}
	return false
}

func pppFilterValue(e equipment.PPPEntry, key string, values map[string]string) string {
	if key == "aps" && e.Kind == "server" {
		if len(e.APs) == 0 {
			return "Sem AP ligado"
		}
		return "Com AP ligado"
	}
	if key == "state" && e.Kind == "secret" && e.State == "" {
		return "Sem estado"
	}
	return values[key]
}

func PPPListCounts(entries []equipment.PPPEntry, screen string) map[string]int {
	if screen != "sessions" {
		return nil
	}
	counts := map[string]int{"ONLINE": 0, "OFFLINE": 0}
	known := false
	for _, e := range entries {
		if e.Kind == "session" {
			known = true
			state := "ONLINE"
			if e.State == "OFFLINE" {
				state = "OFFLINE"
			}
			counts[state]++
		}
	}
	if !known {
		return nil
	}
	return counts
}
