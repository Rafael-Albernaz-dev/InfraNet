package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"time"
)

func (s *Store) SavePulseReading(ctx context.Context, company string, target equipment.PPTarget, reading pulse.Reading) equipment.Code {
	return s.ppWrite(ctx, company, func(tx *sql.Tx) equipment.Code {
		if code := s.verifyMonitoringSaveTarget(ctx, tx, company, target.Device, target.Credential.Metadata.Revision); code != equipment.OK {
			return code
		}
		if err := recordBoot(ctx, tx, target.Device.ID, reading.At, reading.Uptime); err != nil {
			return equipment.Unavailable
		}
		if err := recordIdentity(ctx, tx, target.Device.ID, reading); err != nil {
			return equipment.Unavailable
		}
		if reading.Observation != nil {
			if code := s.persistPPCollection(ctx, tx, reading.Observation); code != equipment.OK {
				return code
			}
		}
		if err := pulseHistoryGC(ctx, tx, reading.At); err != nil {
			return equipment.Unavailable
		}
		return equipment.OK
	})
}

func recordBoot(ctx context.Context, tx *sql.Tx, device string, at time.Time, uptime time.Duration) error {
	if at.IsZero() || uptime < 0 {
		return nil
	}
	boot := pulse.BootTime(at, uptime)
	var saved string
	err := tx.QueryRowContext(ctx, "SELECT boot_at FROM device_boot_checkpoint WHERE device_id=?", device).Scan(&saved)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		previous, err := time.Parse(time.RFC3339Nano, saved)
		if err != nil {
			return err
		}
		if !pulse.Rebooted(previous, boot) {
			return nil
		}
		row := equipment.PPReadRow{Kind: "reboot", At: equipment.PPTime(boot), SourceAt: saved}
		body, err := json.Marshal(row)
		if err != nil {
			return err
		}
		// ignored: Only the SQL error is consumed; the result carries no required domain value.
		// ignored: Only the SQL error is needed; no result value is consumed.
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO device_equipment_events VALUES(?,?,?,?,?,?)",
			device, row.At, boot.Unix(), "reboot", "", string(body)); err != nil {
			return err
		}
	}
	// ignored: Only the SQL error is needed; no result value is consumed.
	_, err = tx.ExecContext(ctx, "INSERT INTO device_boot_checkpoint VALUES(?,?) ON CONFLICT(device_id) DO UPDATE "+
		"SET boot_at=excluded.boot_at", device, equipment.PPTime(boot))
	return err
}

func recordIdentity(ctx context.Context, tx *sql.Tx, device string, reading pulse.Reading) error {
	var firmware, identity, model, stamp string
	err := tx.QueryRowContext(ctx,
		"SELECT firmware,identity,model,at FROM device_identity_checkpoint WHERE device_id=?",
		device).Scan(&firmware,
		&identity,
		&model,
		&stamp)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		previous, parseErr := time.Parse(time.RFC3339Nano, stamp)
		if parseErr != nil {
			return parseErr
		}
		if !reading.At.After(previous) {
			return nil
		}
	}
	if err == nil {
		before := pulse.Identity{Firmware: firmware, Name: identity, Model: model}
		after := pulse.Identity{Firmware: reading.Firmware, Name: reading.Identity, Model: reading.Model}
		for _, v := range pulse.IdentityChanges(before, after) {
			// ignored: Only the SQL error is consumed; the result carries no required domain value.
			// ignored: Only the SQL error is needed; no result value is consumed.
			if _, err = tx.ExecContext(ctx, "INSERT INTO device_pulse_events(device_id,kind,started_at,ended_at,before_value,"+
				"after_value) VALUES(?,?,?,?,?,?)",
				device, v.Kind, equipment.PPTime(reading.At), equipment.PPTime(reading.At), v.Before, v.After); err != nil {
				return err
			}
		}
	}
	// ignored: Only the SQL error is needed; no result value is consumed.
	_, err = tx.ExecContext(ctx, `INSERT INTO device_identity_checkpoint VALUES(?,?,?,?,?) ON CONFLICT(device_id) DO UPDATE SET
 at=excluded.at,firmware=CASE WHEN excluded.firmware='' THEN firmware ELSE excluded.firmware END,
 identity=CASE WHEN excluded.identity='' THEN identity ELSE excluded.identity END,
 model=CASE WHEN excluded.model='' THEN model ELSE excluded.model END`,
		device,
		reading.Firmware,
		reading.Identity,
		reading.Model,
		equipment.PPTime(reading.At))
	return err
}

func (s *Store) MarkPulseGap(ctx context.Context, company string, target equipment.PPTarget, at time.Time) equipment.Code {
	return s.ppWrite(ctx, company, func(tx *sql.Tx) equipment.Code {
		if code := s.verifyMonitoringSaveTarget(ctx, tx, company, target.Device, target.Credential.Metadata.Revision); code != equipment.OK {
			return code
		}
		// ignored: Only the SQL error is consumed; the result carries no required domain value.
		// ignored: Only the SQL error is needed; no result value is consumed.
		_, err := tx.ExecContext(ctx, `INSERT INTO session_collection_checkpoint VALUES(?, '', ?) ON CONFLICT(device_id) DO UPDATE SET
 gap_start=CASE WHEN gap_start='' THEN CASE WHEN last_success='' THEN excluded.gap_start ELSE last_success END ELSE gap_start END`,
			target.Device.ID,
			equipment.PPTime(at))
		if err != nil {
			return equipment.Unavailable
		}
		return equipment.OK
	})
}

func recordManualPulse(ctx context.Context, tx *sql.Tx, d equipment.Device) error {
	r := d.Result
	if r == nil || r.At == "" {
		return nil
	}
	at, err := time.Parse(time.RFC3339Nano, r.At)
	if err != nil {
		return err
	}
	if r.UptimeNS != nil {
		if err = recordBoot(ctx, tx, d.ID, at, time.Duration(*r.UptimeNS)); err != nil {
			return err
		}
	}
	return recordIdentity(ctx, tx, d.ID, pulse.Reading{At: at, Identity: r.Identity, Firmware: r.Version, Model: r.Board})
}

func readRestartSummary(ctx context.Context, db *sql.DB, device string, now time.Time) (equipment.PPReadRow, error) {
	var seven, thirty uint64
	var last string
	err := db.QueryRowContext(ctx, `SELECT COALESCE(sum(CASE WHEN at>=? THEN 1 ELSE 0 END),0),
 COALESCE(sum(CASE WHEN at>=? THEN 1 ELSE 0 END),0),COALESCE(max(stamp),'')
 FROM device_equipment_events WHERE device_id=? AND kind='reboot'`,
		now.Add(-7*24*time.Hour).Unix(),
		now.Add(-30*24*time.Hour).Unix(),
		device).Scan(&seven,
		&thirty,
		&last)
	return equipment.PPReadRow{Kind: "restarts", Used: &seven, Size: &thirty, At: equipment.PPTime(now), SourceAt: last}, err
}
