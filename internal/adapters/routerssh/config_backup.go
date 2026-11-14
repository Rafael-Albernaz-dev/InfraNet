package routerssh

import (
	"context"
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
	"strings"
)

func configCommand(d equipment.Device) string {
	switch cfgbackup.FormatFor(d) {
	case cfgbackup.CFG:
		return "cat /tmp/system.cfg"
	case cfgbackup.RSC:
		version := ""
		if d.Result != nil {
			version = d.Result.Version
		}
		if d.RadioSummary != nil {
			version = d.RadioSummary.Version
		}
		if strings.HasPrefix(version, "7") {
			return "/export show-sensitive"
		}
		return "/export"
	default:
		return ""
	}
}
func (Adapter) ReadConfig(ctx context.Context, d equipment.Device, user, secret string,
	check func() equipment.Code) (string, equipment.Code) {
	command := configCommand(d)
	if command == "" {
		return "", cfgbackup.Unsupported
	}
	connection, code := openPPConnection(ctx, d, user, secret, check, equipment.QueryBudget)
	secret = ""
	if code != equipment.OK {
		return "", code
	}
	defer connection.close()
	connection.configBudget()
	text, code := connection.run(command)
	if code != equipment.OK {
		return "", code
	}
	if code = check(); code != equipment.OK {
		return "", code
	}
	if strings.TrimSpace(text) == "" || !strings.HasSuffix(text, "\n") {
		return "", equipment.Partial
	}
	return text, equipment.OK
}
