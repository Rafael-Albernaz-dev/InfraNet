package pppoe

import (
	"time"
)

const UnknownDisconnectReason = "Não informado nesta leitura"

func SessionKey(row PPRow) string {
	if row.RemoteID == "" {
		return "row:" + row.ID
	}
	return row.RemoteID + "\x00" + row.SessionID + "\x00" + row.Login
}

func ConnectedAt(row PPRow, at time.Time) string {
	if row.Uptime.Known {
		return PPTime(at.Add(-time.Duration(row.Uptime.Min)))
	}
	return PPTime(at)
}

// AverageSessions associates counters by the interface's canonical PPP name, once per cycle.
func AverageSessions(rows []PPRow, previous, current []PPTunnel) {
	before := map[string]PPTunnel{}
	counts := map[string]int{}
	for _, tunnel := range previous {
		before[tunnel.Name] = tunnel
	}
	for _, row := range rows {
		counts[row.Login]++
	}
	by := map[string]PPTunnel{}
	for _, tunnel := range current {
		by[tunnel.Name] = tunnel
	}
	for i := range rows {
		r := &rows[i]
		r.Download, r.Upload = nil, nil
		if r.State == "OFFLINE" || counts[r.Login] != 1 {
			continue
		}
		name := "<pppoe-" + r.Login + ">"
		if r.Context != nil && r.Context.Interface != "" {
			name = r.Context.Interface
		}
		now, ok := by[name]
		old, had := before[name]
		if ok && r.Context == nil && counts[r.Login] == 1 {
			r.Context = &PPContext{Association: "associated", Interface: name}
		}
		if ok {
			r.MTU = now.MTU
		}
		if !ok || !had {
			continue
		}
		r.Download, r.Upload = PPBandwidth(old, now)
	}
}
