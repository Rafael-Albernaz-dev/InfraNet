package storage

import (
	"context"
	"database/sql"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"time"
)

func (s *Store) LoadPresence(ctx context.Context, company string) (map[string]pulse.DeviceStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]pulse.DeviceStatus{}
	if s.unit == nil || s.unitID != company {
		return out, nil
	}
	rows, err := s.unit.QueryContext(ctx, "SELECT device_id,address,last_online_at,state,since FROM device_presence")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, reply, since string
		var value pulse.DeviceStatus
		if err = rows.Scan(&id, &value.Address, &reply, &value.State, &since); err != nil {
			return nil, err
		}
		if reply != "" {
			value.LastReply, err = time.Parse(time.RFC3339Nano, reply)
			if err != nil {
				return nil, err
			}
		}
		if since != "" {
			value.Since, err = time.Parse(time.RFC3339Nano, since)
			if err != nil {
				return nil, err
			}
		}
		out[id] = value
	}
	return out, rows.Err()
}

func historyTime(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return equipment.PPTime(at)
}

func (s *Store) SavePresence(ctx context.Context, company string, values map[string]pulse.DeviceStatus, at time.Time) error {
	var failure error
	code := s.ppWrite(ctx, company, func(tx *sql.Tx) equipment.Code {
		for id, value := range values {
			if err := savePresence(ctx, tx, id, value, at); err != nil {
				failure = err
				return equipment.Unavailable
			}
		}
		if err := pulseHistoryGC(ctx, tx, at); err != nil {
			failure = err
			return equipment.Unavailable
		}
		return equipment.OK
	})
	if failure != nil {
		return failure
	}
	if code != equipment.OK {
		return errPresenceUnavailable
	}
	return nil
}

var errPresenceUnavailable = &presenceError{}

type presenceError struct{}

func (*presenceError) Error() string { return "presence persistence unavailable" }
func savePresence(ctx context.Context, tx *sql.Tx, id string, value pulse.DeviceStatus, at time.Time) error {
	var address, reply, since string
	var state pulse.State
	var saved int64
	err := tx.QueryRowContext(ctx, "SELECT address,last_online_at,state,since,saved_at FROM device_presence WHERE device_id=?", id).
		Scan(&address, &reply, &state, &since, &saved)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	previous := pulse.DeviceStatus{Address: address, State: state}
	changed := pulse.PresenceChanged(previous, value)
	if !pulse.PresenceDue(previous, value, time.Unix(saved, 0), at) {
		return nil
	}
	if changed {
		start := historyTime(value.Since)
		// ignored: Only the SQL error is consumed; the result carries no required domain value.
		// ignored: Only the SQL error is needed; no result value is consumed.
		if _, err = tx.ExecContext(ctx, "UPDATE device_pulse_events SET ended_at=? WHERE device_id=? AND kind IN ('ONLINE"+
			"','OFFLINE') AND ended_at=''", start, id); err != nil {
			return err
		}
		// ignored: Only the SQL error is needed; no result value is consumed.
		if _, err = tx.ExecContext(ctx,
			"INSERT INTO device_pulse_events(device_id,kind,started_at) VALUES(?,?,?)",
			id,
			value.State,
			start); err != nil {
			return err
		}
	}
	// ignored: Only the SQL error is needed; no result value is consumed.
	_, err = tx.ExecContext(ctx, `INSERT INTO device_presence VALUES(?,?,?,?,?,?) ON CONFLICT(device_id) DO UPDATE SET
 address=excluded.address,last_online_at=excluded.last_online_at,state=excluded.state,since=excluded.since,saved_at=excluded.saved_at`,
		id, value.Address, historyTime(value.LastReply), value.State, historyTime(value.Since), at.Unix())
	return err
}

func bumpSessionHistory(ctx context.Context, tx *sql.Tx) error {
	// ignored: Only the statement error is relevant for this counter.
	_, err := tx.ExecContext(ctx, "UPDATE session_history_generation SET generation=generation+1 WHERE singleton=1")
	return err
}
func pulseHistoryGC(ctx context.Context, tx *sql.Tx, at time.Time) error {
	cut := equipment.PPTime(at.Add(-pulse.HistoryRetention))
	result, err := tx.ExecContext(ctx, "DELETE FROM session_history WHERE disconnected_at<>'' AND disconnected_at<?", cut)
	if err != nil {
		return err
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if removed > 0 {
		if err = bumpSessionHistory(ctx, tx); err != nil {
			return err
		}
	}
	// ignored: Only the statement error is relevant to retention.
	if _, err = tx.ExecContext(ctx, "DELETE FROM device_pulse_events WHERE started_at<? AND ended_at<>''", cut); err != nil {
		return err
	}
	// ignored: Only the statement error is relevant to retention.
	_, err = tx.ExecContext(ctx, "DELETE FROM device_equipment_events WHERE at<?", at.Add(-pulse.HistoryRetention).Unix())
	return err
}
