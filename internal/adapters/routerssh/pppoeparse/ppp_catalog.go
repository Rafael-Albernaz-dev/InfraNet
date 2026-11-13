package pppoeparse

import (
	"infradesktop/internal/adapters/routerssh/internal/routeros"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pppoe"
	"strconv"
	"strings"
)

var PPPCatalogCommand = buildPPPCatalog()

var catalogFields = map[string][]string{
	"secret": {"name", "service", "profile", "disabled", "comment", "caller-id", "remote-address",
		"last-logged-out", "last-caller-id", "last-disconnect-reason", "local-address"},
	"vlan":    {"name", "vlan-id", "interface"},
	"profile": {"name", "rate-limit", "local-address", "remote-address", "only-one"},
	"server":  {"service-name", "interface", "disabled", "invalid", "max-mtu", "max-mru", "default-profile", "authentication"},
}

func buildPPPCatalog() string {
	s := routeros.EscapeScript + `; :local n 0; :local bytes 0; :local limited false; :put "P\t1"; `
	for _, kind := range []string{"secret", "profile", "server", "vlan"} {
		path := "/ppp " + kind
		if kind == "vlan" {
			path = "/interface vlan"
		}
		if kind == "server" {
			path = "/interface pppoe-server server"
		}
		s += `:local ids [` + path + ` find]; :local count 0; :foreach i in=$ids do={:if ($count<4096) do={`
		expr := `"` + kind + `\t".[$esc $i]`
		for j, field := range catalogFields[kind] {
			variable := "v" + strconv.Itoa(j)
			s += `:local ` + variable + ` "?"; :do {:set ` + variable + ` ("=".[:pick [:tostr [` +
				path + ` get $i ` + field + `]] 0 256])} on-error={}; `
			expr += `."\t".[$esc $` + variable + `]`
		}
		s += `:local line (` + expr + `); :if (($bytes+[:len $line])<3145728) do={:put $line; :set n ($n+1); ` +
			`:set bytes ($bytes+[:len $line]+1)} else={:set limited true}; ` +
			`:set count ($count+1)} else={:set limited true}}; `
	}
	return s + `:put ("Z\t".$n."\t".$limited)`
}

func ParsePPPCatalog(text string) ([]equipment.PPReadRow, bool, equipment.Code) {
	if len(text) > equipment.PPPCatalogBytes {
		return nil, false, equipment.Limit
	}

	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	limited, valid := catalogEnvelope(lines)
	if !valid {
		return nil, false, equipment.Partial
	}
	out := []equipment.PPReadRow{}
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, line := range lines[1 : len(lines)-1] {
		e, ok := catalogLine(line)
		if !ok {
			return nil, false, equipment.Partial
		}
		key := e.Kind + "/" + e.ID
		if seen[key] || counts[e.Kind] >= equipment.PPPMaxSecrets {
			return nil, false, equipment.Partial
		}
		seen[key] = true
		counts[e.Kind]++
		out = append(out, equipment.PPReadRow{Kind: e.Kind, Entry: &e})
	}
	return out, limited, equipment.OK
}
func catalogEnvelope(lines []string) (bool, bool) {
	if len(lines) < 2 || strings.TrimSuffix(lines[0], "\r") != "P\t1" {
		return false, false
	}
	f := strings.Split(strings.TrimSuffix(lines[len(lines)-1], "\r"), "\t")
	if len(f) != 3 || f[0] != "Z" {
		return false, false
	}
	n, err := strconv.Atoi(f[1])
	return f[2] == "true", err == nil && n == len(lines)-2 && (f[2] == "true" || f[2] == "false")
}
func catalogLine(line string) (equipment.PPPEntry, bool) {
	f := strings.Split(strings.TrimSuffix(line, "\r"), "\t")
	if len(f) < 2 || len(catalogFields[f[0]]) == 0 || len(f) != len(catalogFields[f[0]])+2 {
		return equipment.PPPEntry{}, false
	}
	for i := 1; i < len(f); i++ {
		value, ok := routeros.Unescape(f[i])
		if !ok || len(value) > 257 {
			return equipment.PPPEntry{}, false
		}
		f[i] = value
	}
	if f[1] == "" {
		return equipment.PPPEntry{}, false
	}
	return catalogEntry(f), true
}

func catalogEntry(f []string) equipment.PPPEntry {
	values := map[string]string{}
	for i, field := range catalogFields[f[0]] {
		if field == "comment" && strings.HasPrefix(f[i+2], "=") {
			values[field] = f[i+2][1:]
		} else if value := ppSecretText(f[i+2], 256); value != nil {
			values[field] = *value
		}
	}
	e := equipment.PPPEntry{ID: f[1], Kind: f[0], Name: values["name"], Service: values["service"], Profile: values["profile"],
		Client: pppoe.ClientName(values["comment"]), Caller: values["caller-id"], RemoteAddress: values["remote-address"],
		LastOut: values["last-logged-out"], LastCaller: values["last-caller-id"], LastReason: values["last-disconnect-reason"],
		Rate: values["rate-limit"], LocalAddress: values["local-address"], OnlyOne: values["only-one"],
		Interface: values["interface"], MTU: values["max-mtu"], MRU: values["max-mru"], Authentication: values["authentication"]}
	if f[0] == "vlan" {
		if n, err := strconv.Atoi(values["vlan-id"]); err == nil && n > 0 && n < 4095 {
			e.VLAN = &n
		}
	}
	if f[0] == "server" {
		e.Name = values["service-name"]
		e.Service = e.Name
		e.Profile = values["default-profile"]
	}
	if value, ok := values["disabled"]; ok {
		e.Disabled = pdBoolPointer(value)
	}
	if value, ok := values["invalid"]; ok {
		e.Invalid = pdBoolPointer(value)
	}
	return e
}

func pdBoolPointer(value string) *bool {
	if value != "true" && value != "false" && value != "yes" && value != "no" {
		return nil
	}
	b := value == "true" || value == "yes"
	return &b
}
