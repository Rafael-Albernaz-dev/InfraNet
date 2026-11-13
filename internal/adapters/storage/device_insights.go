package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"time"
)

type equipmentInsights struct {
	device     equipment.Device
	credential uint64
	result     *equipment.PPReadResult
	health     *equipment.DeviceHealth
	at         time.Time
}

func saveEquipmentInsights(ctx context.Context, tx *sql.Tx, in equipmentInsights) error {
	previous, err := priorEquipmentPorts(ctx, tx, in)
	if err != nil || previous.skip {
		return err
	}
	if in.health.Uptime.Known {
		at, err := time.Parse(time.RFC3339Nano, in.health.At)
		if err != nil {
			return err
		}
		if err = recordBoot(ctx, tx, in.device.ID, at, time.Duration(in.health.Uptime.Min)); err != nil {
			return err
		}
	}
	ports, err := recordPortChanges(ctx, tx, in, previous)
	if err != nil {
		return err
	}
	return writeEquipmentSnapshot(ctx, tx, in, ports)
}

func readEquipmentInsights(ctx context.Context, db *sql.DB, device, scope string, now time.Time) ([]equipment.PPReadRow, error) {
	rows := []equipment.PPReadRow{}
	if scope == "port-history" {
		q, e := db.QueryContext(ctx, "SELECT row_json FROM device_equipment_events WHERE device_id=? AND kind='port_change' AND at>=? ORDER BY at DESC,stamp DESC,name LIMIT 2049", device, now.Add(-30*24*time.Hour).Unix())
		if e != nil {
			return nil, e
		}
		defer q.Close()
		for q.Next() {
			var raw string
			var r equipment.PPReadRow
			if e = q.Scan(&raw); e != nil {
				return nil, e
			}
			if e = json.Unmarshal([]byte(raw), &r); e != nil {
				return nil, e
			}
			rows = append(rows, r)
		}
		return rows, q.Err()
	}
	restarts, err := readRestartSummary(ctx, db, device, now)
	if err != nil {
		return nil, err
	}
	rows = append(rows, restarts)
	var raw, ports, stamp string
	e := db.QueryRowContext(ctx, "SELECT s.health_json,s.ports_json,s.stamp FROM device_equipment_snapshot s JOIN "+
		"devices d ON d.id=s.device_id WHERE s.device_id=? AND s.revision=d.revision", device).Scan(&raw, &ports, &stamp)
	if e == sql.ErrNoRows {
		if restarts.SourceAt == "" {
			return []equipment.PPReadRow{}, nil
		}
		return rows, nil
	}
	if e != nil {
		return nil, e
	}
	var h equipment.DeviceHealth
	if e = json.Unmarshal([]byte(raw), &h); e != nil {
		return nil, e
	}
	var portRows []equipment.PPReadRow
	if e = json.Unmarshal([]byte(ports), &portRows); e != nil {
		return nil, e
	}
	rows = append(rows, equipment.PPReadRow{Kind: "resource", Health: &h, At: stamp})
	rows = append(rows, portRows...)
	return rows, nil
}
