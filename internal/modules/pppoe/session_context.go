package pppoe

import (
	"strings"
	"time"
	"unicode"
)

// CarryContext joins only a continuing remote session, never a login on its own.
// A minimal counter projection is not a new context reading.
func CarryContext(previous, current []PPRow) {
	by := map[string]*PPContext{}
	for _, row := range previous {
		if row.RemoteID != "" && row.SessionID != "" {
			by[SessionKey(row)] = row.Context
		}
	}
	for i := range current {
		row := &current[i]
		if row.RemoteID == "" || row.SessionID == "" || HasSessionContext(row.Context) {
			continue
		}
		old := by[SessionKey(*row)]
		if old == nil {
			continue
		}
		copy := *old
		if old.VLAN != nil {
			n := *old.VLAN
			copy.VLAN = &n
		}
		if row.Context != nil {
			copy.Association = row.Context.Association
			if row.Context.Interface != "" {
				copy.Interface = row.Context.Interface
			}
		}
		row.Context = &copy
	}
}

func HasSessionContext(c *PPContext) bool {
	return c != nil && (c.SecretState != "" || c.Profile != "" || c.Service != "" || c.VLAN != nil)
}

func NeedsSessionContext(rows []PPRow, last, now time.Time) bool {
	if !last.IsZero() && now.Sub(last) < 5*time.Minute {
		return false
	}
	for _, row := range rows {
		if row.State != "OFFLINE" && !HasSessionContext(row.Context) {
			return true
		}
	}
	return false
}

// ClientName remains local to the operator's company. Normalize whitespace
// before applying the display ceiling so control characters never reach a cell.
func ClientName(comment string) string {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, comment)
	clean = strings.TrimSpace(clean)
	if len(clean) >= 4 && strings.EqualFold(clean[:4], "SGP:") {
		clean = clean[4:]
	}
	clean = strings.Join(strings.Fields(clean), " ")
	runes := []rune(clean)
	if len(runes) > 120 {
		return string(runes[:120])
	}
	return clean
}

// Session IDs can be reused after reboot. A regressed or unknown router clock
// cannot justify inheriting a plan from an earlier router session.
func CarryObservationContext(previous, current *PPObservation) {
	if previous == nil || current == nil {
		return
	}
	a, b := previous.Provenance, current.Provenance
	if a.Version != b.Version || !a.RouterUptime.Known || !b.RouterUptime.Known || b.RouterUptime.Max < a.RouterUptime.Min {
		return
	}
	CarryContext(previous.Rows, current.Rows)
}
