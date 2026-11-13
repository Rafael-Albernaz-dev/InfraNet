package ipc

import (
	"encoding/base64"
	"infradesktop/internal/modules/equipment"
)

func sanitizePPExport(v *equipment.PPView, input equipment.PPInput) equipment.Code {
	key, err := base64.RawStdEncoding.DecodeString(input.ExportKey)
	if err != nil {
		return equipment.Invalid
	}
	for _, row := range v.DiagnosticRows {
		v.ExportDiagnostics = append(v.ExportDiagnostics, sanitizedDiagnosticRow(key, input.Scope, v.Diagnostic.ID, row))
	}
	for _, row := range v.Rows {
		v.ExportRows = append(v.ExportRows, sanitizedSessionRow(key, v.Current.ID, row))
	}
	for _, delta := range v.Deltas {
		v.ExportDeltas = append(v.ExportDeltas, sanitizedDelta(key, input, delta))
	}
	v.Deltas = nil
	v.Rows = nil
	v.Current = nil
	v.Diagnostic = nil
	v.DiagnosticRows = nil
	return equipment.OK
}
func sanitizedDiagnosticRow(key []byte, scope, id string, row equipment.PDRow) equipment.PDRow {

	row.ID = ppToken(key, "diagnostic-row", id+row.ID)
	if row.RemoteID != "" {
		row.RemoteID = ppToken(key, "diagnostic-remote", scope+"/"+row.RemoteID)
	}
	row.Name = ppToken(key, "diagnostic-name", row.Name)
	row.Parent = ppToken(key, "diagnostic-name", row.Parent)
	if row.ParentID != "" {
		row.ParentID = ppToken(key, "diagnostic-row", id+row.ParentID)
	}
	row.Address = ppToken(key, "address", row.Address)
	switch row.Type {
	case "", "ether", "vlan", "bridge", "pppoe-in", "pppoe-out", "wlan", "wifi",
		"bonding", "eoip", "veth", "loopback", "gre", "vxlan", "wireguard":
	default:
		row.Type = "unknown"
	}
	return row

}
func sanitizedSessionRow(key []byte, observation string, row equipment.PPRow) equipment.PPExportRow {
	if row.State == "" {
		row.State = "ONLINE"
	}
	return equipment.PPExportRow{State: row.State,
		Uptime: row.Uptime, Duplicates: row.Duplicates,
		Observation: ppToken(key, "observation", observation),
		Row:         ppToken(key, "row", observation+row.ID),
		Login:       ppToken(key, "login", row.Login),
		Address:     ppToken(key, "address", row.Address),
		Caller:      ppToken(key, "caller", row.Caller)}

}
func sanitizedDelta(key []byte, input equipment.PPInput, delta equipment.PPDelta) equipment.PPExportDelta {
	return equipment.PPExportDelta{BaseRow: ppToken(key, "row", func() string {
		if delta.BaseRow == "" {
			return ""
		}
		return input.BaseID + delta.BaseRow
	}()), TargetRow: ppToken(key, "row", func() string {
		if delta.TargetRow == "" {
			return ""
		}
		return input.TargetID + delta.TargetRow
	}()), Category: delta.Category, Reason: delta.Reason,
		AddressChanged: delta.AddressChanged, CallerChanged: delta.CallerChanged}

}
