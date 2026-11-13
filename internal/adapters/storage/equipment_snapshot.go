package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/monitoring"
	"time"
)

type previousEquipmentPorts struct {
	values map[string]*equipment.DevicePort
	stamp  string
	skip   bool
}

func priorEquipmentPorts(ctx context.Context, tx *sql.Tx, in equipmentInsights) (previousEquipmentPorts, error) {
	old := previousEquipmentPorts{values: map[string]*equipment.DevicePort{}}
	var revision, credential uint64
	var connection, health, ports string
	query := `SELECT revision,credential_revision,connection,stamp,health_json,ports_json
 FROM device_equipment_snapshot WHERE device_id=?`
	err := tx.QueryRowContext(ctx, query, in.device.ID).Scan(&revision, &credential, &connection, &old.stamp, &health, &ports)
	if err == sql.ErrNoRows {
		return old, nil
	}
	if err != nil {
		return old, err
	}
	at, err := time.Parse(time.RFC3339Nano, old.stamp)
	if err != nil {
		return old, err
	}
	old.skip = !in.at.After(at)
	if old.skip || revision != in.device.Revision || credential != in.credential || connection != equipment.PPConnection(in.device) {
		return old, nil
	}
	var previous equipment.DeviceHealth
	if err = json.Unmarshal([]byte(health), &previous); err != nil {
		return old, err
	}
	var rows []equipment.PPReadRow
	if err = json.Unmarshal([]byte(ports), &rows); err != nil {
		return old, err
	}
	for _, row := range rows {
		old.values[row.Name] = row.Port
	}
	return old, nil
}

func recordPortChanges(ctx context.Context, tx *sql.Tx, in equipmentInsights, old previousEquipmentPorts) ([]equipment.PPReadRow, error) {
	ports := []equipment.PPReadRow{}
	if in.result.Availability["ports"] != "complete" {
		return ports, nil
	}
	stamp := equipment.PPTime(in.at)
	for _, row := range in.result.Rows {
		if row.Kind != "port" || row.Port == nil {
			continue
		}
		row.At = stamp
		ports = append(ports, row)
		before := old.values[row.Name]
		if !monitoring.PortStatusChanged(before, row.Port) {
			continue
		}
		event := row
		event.Kind, event.BeforePort, event.SourceAt = "port_change", before, old.stamp
		body, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		// ignored: Only the SQL error is needed for the atomic write.
		// ignored: Only the SQL error is needed; no result value is consumed.
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO device_equipment_events VALUES(?,?,?,?,?,?)",
			in.device.ID, stamp, in.at.Unix(), "port_change", row.Name, string(body))
		if err != nil {
			return nil, err
		}
	}
	return ports, nil
}

func writeEquipmentSnapshot(ctx context.Context, tx *sql.Tx, in equipmentInsights, ports []equipment.PPReadRow) error {
	body, err := json.Marshal(in.health)
	if err != nil {
		return err
	}
	portBody, err := json.Marshal(ports)
	if err != nil {
		return err
	}
	query := `INSERT INTO device_equipment_snapshot VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(device_id) DO UPDATE SET
 revision=excluded.revision,credential_revision=excluded.credential_revision,connection=excluded.connection,
 at=excluded.at,stamp=excluded.stamp,health_json=excluded.health_json,ports_json=excluded.ports_json`
	// ignored: Only the SQL error is needed for the atomic write.
	// ignored: Only the SQL error is needed; no result value is consumed.
	_, err = tx.ExecContext(ctx, query, in.device.ID, in.device.Revision, in.credential, equipment.PPConnection(in.device),
		in.at.Unix(), equipment.PPTime(in.at), string(body), string(portBody))
	return err
}
