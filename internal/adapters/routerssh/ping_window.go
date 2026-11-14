package routerssh

import (
	"context"
	"infradesktop/internal/adapters/routerssh/pppoeparse"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pppoe"
	"strings"
	"time"
)

func readPingWindow(ctx context.Context, req equipment.PPReadRequest) (*equipment.PPReadResult, equipment.Code) {
	if req.Target == nil || req.Target.Ping == nil {
		return nil, equipment.Invalid
	}
	target := req.Target.Ping
	// ignored: validate the command before opening SSH; rebuild after the prelude.
	if _, ok := pppoeparse.BuildPingWindowCommand(target); !ok {
		return nil, equipment.ObservationIncompatible
	}
	connection, code := openPPConnection(ctx, req.Device, req.User, req.Secret, req.Check, equipment.PingWindowBudget)
	req.Secret = ""
	if code != equipment.OK {
		return nil, code
	}
	defer connection.close()
	if code = checkPingWindowContext(connection, target); code != equipment.OK {
		return nil, code
	}
	if code = req.Check(); code != equipment.OK {
		return nil, code
	}
	command, ok := pppoeparse.BuildPingWindowCommand(target)
	if !ok {
		return nil, equipment.ObservationIncompatible
	}
	window := &pppoe.PingWindow{Started: time.Now()}
	code = connection.stream(command, func(line string) equipment.Code { return publishPingWindowLine(req, window, line) })
	result, measured := ppMeasureRead(window.Result(time.Now()))
	if code == equipment.OK {
		code = measured
	}
	return result, code
}
func checkPingWindowContext(connection *ppConnection, target *equipment.PPPingTarget) equipment.Code {
	prelude := pppoeparse.PDPingContextV7
	family := target.Observation.Provenance.Family
	if family == equipment.RouterFamilyV6 {
		prelude = pppoeparse.PDPingContextV6
	}
	text, code := connection.run(prelude)
	if code != equipment.OK {
		return code
	}
	if !pppoeparse.PingContextSafe(family, text) {
		return "PING_CONTEXT_UNAVAILABLE"
	}
	return equipment.OK
}
func publishPingWindowLine(req equipment.PPReadRequest, window *pppoe.PingWindow, line string) equipment.Code {
	if window.Expired(time.Now()) {
		return equipment.RadioCodeDeadlineExceeded
	}
	clean := strings.TrimSpace(line)
	switch clean {
	case "E\tcontext":
		return "PING_CONTEXT_UNAVAILABLE"
	case "E\tsession":
		return "PING_SESSION_CHANGED"
	}
	if strings.HasPrefix(clean, "SEQ ") || strings.HasPrefix(clean, "sent=") {
		return equipment.OK
	}
	probe, ok := pppoeparse.ParsePingWindowLine(line, req.Target.Ping.Row.Address)
	if !ok {
		return equipment.Partial
	}
	if code := window.Add(line, probe); code != equipment.OK {
		return code
	}
	result, code := ppMeasureRead(window.Result(time.Now()))
	if code != equipment.OK {
		return code
	}
	return req.Publish(result)
}
