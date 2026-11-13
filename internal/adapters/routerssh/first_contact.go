package routerssh

import (
	"context"
	"infradesktop/internal/adapters/routerssh/radioparse"
	"infradesktop/internal/modules/equipment"
	"strings"
	"time"
)

func (a Adapter) FirstContact(ctx context.Context, d equipment.Device, user, secret string,
	guard func() equipment.Code) (*equipment.ContactResult, equipment.Code) {
	if code := guard(); code != equipment.OK {
		return nil, code
	}
	presented, code := a.Inspect(ctx, d)
	if code != equipment.OK {
		return nil, code
	}
	d.Fingerprint, code = equipment.FirstContactKey(d.Fingerprint, presented)
	if code != equipment.OK {
		return nil, code
	}
	conn, done, code := dial(ctx, d)
	if code != equipment.OK {
		return nil, code
	}
	defer done()
	client, code := openQuerySSHClient(ctx, conn, d, user, secret, guard)
	secret = ""
	if code != equipment.OK {
		return nil, code
	}
	defer client.Close()
	budget := &outputBudget{conn: conn}
	run := func(command string) (string, equipment.Code) {
		return runQuerySession(ctx, client, budget, command, guard)
	}
	var observation *equipment.Observation
	if strings.HasPrefix(string(client.ServerVersion()), "SSH-2.0-ROSSSH") {
		observation, code = readQuerySummary(d, run)
	} else {
		observation, code = contactRadioSummary(run)
	}
	if code != equipment.OK {
		return nil, code
	}
	return &equipment.ContactResult{Fingerprint: d.Fingerprint, Observation: observation}, equipment.OK
}

func contactRadioSummary(run func(string) (string, equipment.Code)) (*equipment.Observation, equipment.Code) {
	const command = `if command -v mca-status >/dev/null 2>&1; then mca-status; cat /proc/uptime; else printf "L07-DRIVER-ABSENT\n"; fi`
	text, code := run(command)
	if code != equipment.OK {
		return nil, code
	}
	at := time.Now().UTC().Format(time.RFC3339Nano)
	reading, valid := radioparse.IdentifyAirOS(text)
	if !valid {
		// Key pinned and password accepted: the device is reachable even when no
		// driver reads its name (Intelbras WOM, Mimosa, switch); the name stays the IP.
		return &equipment.Observation{At: at, Quality: "partial", Source: "SSH"}, equipment.OK
	}
	return &equipment.Observation{Identity: reading.Identity, Board: reading.Model, Version: reading.Version,
		Family: string(reading.Family), At: at, Quality: "complete", Source: "SSH"}, equipment.OK
}
