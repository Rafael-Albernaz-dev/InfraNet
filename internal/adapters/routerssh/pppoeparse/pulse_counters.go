package pppoeparse

import (
	"infradesktop/internal/adapters/routerssh/internal/routeros"
	"infradesktop/internal/modules/equipment"
	"strings"
)

// One linear scan; no monitor and no name lookup inside the loop. A session that
// disconnects during the scan removes its interface: that item is skipped, never
// the whole scan, and its average waits for the next cycle.
const PulseCountersCommand = routeros.EscapeScript + `; :local ids [/interface find where type="pppoe-in"]; ` +
	`:local n 0; :if ([:len $ids]>2048) do={:error "rows"}; :foreach i in=$ids do={:do {` +
	`:local line ("C\t".[$esc [/interface get $i name]]."\t".[/interface get $i rx-byte]."\t".` +
	`[/interface get $i tx-byte]); ` +
	`:put $line; :set n ($n+1)} on-error={}}; :put ("Z\t".$n)`

func ParsePulseCounters(text string) ([]equipment.PPTunnel, equipment.Code) {
	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	if len(text) > equipment.PPWireBytes || len(lines) > equipment.PPMaxRows+1 {
		return nil, equipment.Limit
	}
	out := []equipment.PPTunnel{}
	seen := map[string]bool{}
	for i, line := range lines {
		f := strings.Split(strings.TrimSuffix(line, "\r"), "\t")
		if i == len(lines)-1 {
			if len(f) != 2 || f[0] != "Z" {
				return nil, equipment.Partial
			}
			n, ok := pdNumber(f[1], equipment.PPMaxRows)
			if !ok || int(n) != len(out) {
				return nil, equipment.Partial
			}
			return out, equipment.OK
		}
		if len(f) != 4 || f[0] != "C" {
			return nil, equipment.Partial
		}
		row, ok := parsePulseCounter(f)
		if !ok || seen[row.Name] {
			return nil, equipment.Partial
		}
		seen[row.Name] = true
		out = append(out, row)
	}
	return nil, equipment.Partial
}

func parsePulseCounter(f []string) (equipment.PPTunnel, bool) {
	name, ok := routeros.Unescape(f[1])
	rx, tx := routeros.Counter(f[2]), routeros.Counter(f[3])
	if !ok || name == "" || len(name) > 256 || rx == nil || tx == nil {
		return equipment.PPTunnel{}, false
	}
	return equipment.PPTunnel{ID: name, Name: name, RX: rx, TX: tx}, true
}
