package deviceparse

import (
	"infradesktop/internal/adapters/routerssh/internal/routeros"
	"infradesktop/internal/adapters/routerssh/pppoeparse"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"strconv"
	"strings"
	"time"
)

const liveCommandPrefix = `:put ("L\t1\t".[/system resource get uptime]."\t".[/system resource get cpu-load]."\t".`
const LiveConcentratorScript = liveCommandPrefix + "[:len [/ppp active find where service=pppoe]].\"\\t\".[/system identity get name].\"" +
	"\\t\".[/system resource get version].\"\\t\".[/system resource get board-name])"
const LiveAPScript = liveCommandPrefix + "[:len [/interface wireless registration-table find]].\"\\t\".[/system identity get " +
	"name].\"\\t\".[/system resource get version].\"\\t\".[/system resource get board-name]" +
	")"

func ParseLiveReading(output string) (pulse.Reading, pulse.ReadFault) {
	fields := strings.Split(strings.TrimRight(output, "\r\n"), "\t")
	if (len(fields) != 5 && len(fields) != 6 && len(fields) != 8) || fields[0] != "L" || fields[1] != "1" {
		return pulse.Reading{}, pulse.ReadLost
	}
	uptime := pppoeparse.ParsePPDuration(fields[2])
	cpu, errCPU := strconv.Atoi(fields[3])
	clients, errClients := strconv.Atoi(fields[4])
	if !uptime.Known || !routeros.Decimal(fields[3]) || !routeros.Decimal(fields[4]) ||
		errCPU != nil || errClients != nil || cpu < 0 || cpu > 100 || clients < 0 {
		return pulse.Reading{}, pulse.ReadLost
	}
	identity, firmware, model, ok := parseLiveIdentity(fields)
	if !ok {
		return pulse.Reading{}, pulse.ReadLost
	}
	return pulse.Reading{Identity: identity,
			Firmware: firmware,
			Model:    model,
			CPU:      cpu,
			Clients:  clients,
			Uptime:   time.Duration(uptime.Min)},
		pulse.ReadOK
}

func parseLiveIdentity(fields []string) (identity, firmware, model string, ok bool) {
	if len(fields) >= 6 {
		if !equipment.ValidText(fields[5], 128) {
			return
		}
		identity = fields[5]
	}
	if len(fields) == 8 {
		if !equipment.ValidText(fields[6], 128) || !equipment.ValidText(fields[7], 128) {
			return
		}
		firmware, model = fields[6], fields[7]
	}
	return identity, firmware, model, true
}
