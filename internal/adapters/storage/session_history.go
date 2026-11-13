package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pppoe"
	"infradesktop/internal/modules/pulse"
	"time"
)

func recordSessionHistory(ctx context.Context, tx *sql.Tx, o *equipment.PPObservation) error {
	at, err := time.Parse(time.RFC3339Nano, o.Provenance.Ended)
	if err != nil {
		return err
	}
	gap, err := closeCollectionGap(ctx, tx, o.DeviceID, at)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,session_key,login FROM session_history WHERE device_id=? AND kind='ses"+
		"sion' AND disconnected_at=''", o.DeviceID)
	if err != nil {
		return err
	}
	open := []pppoe.OpenSession{}
	for rows.Next() {
		var item pppoe.OpenSession
		if err = rows.Scan(&item.ID, &item.Key, &item.Login); err != nil {
			rows.Close()
			return err
		}
		open = append(open, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	plan := pppoe.ReconcileHistory(open, o.Rows, gap)
	if err = applySessionHistoryPlan(ctx, tx, o, plan); err != nil {
		return err
	}
	if gap || len(plan.Connect) > 0 || len(plan.Disconnect) > 0 || len(plan.Rebind) > 0 {
		if err := bumpSessionHistory(ctx, tx); err != nil {
			return err
		}
	}
	return pulseHistoryGC(ctx, tx, at)
}

func closeCollectionGap(ctx context.Context, tx *sql.Tx, device string, at time.Time) (bool, error) {
	var last, gap string
	err := tx.QueryRowContext(ctx,
		"SELECT last_success,gap_start FROM session_collection_checkpoint WHERE device_id=?",
		device).Scan(&last,
		&gap)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	lastAt, gapAt := time.Time{}, time.Time{}
	if last != "" {
		lastAt, err = time.Parse(time.RFC3339Nano, last)
		if err != nil {
			return false, err
		}
	}
	if gap != "" {
		gapAt, err = time.Parse(time.RFC3339Nano, gap)
		if err != nil {
			return false, err
		}
	}
	gap = historyTime(pulse.CollectionGap(lastAt, gapAt, at))
	if gap != "" {
		// ignored: Only the SQL error is consumed; the result carries no required domain value.
		// ignored: Only the SQL error is needed; no result value is consumed.
		if _, err = tx.ExecContext(ctx, "INSERT INTO session_history(device_id,login,session_key,kind,connected_at,discon"+
			"nected_at,reason,row_json) VALUES(?,'','','gap',?,?,'Lacuna de coleta','{}')", device, gap, equipment.PPTime(at)); err != nil {
			return false, err
		}
	}
	// ignored: Only the SQL error is needed; no result value is consumed.
	_, err = tx.ExecContext(ctx, "INSERT INTO session_collection_checkpoint VALUES(?,?,'') ON CONFLICT(device_id) "+
		"DO UPDATE SET last_success=excluded.last_success,gap_start=''", device, equipment.PPTime(at))
	return gap != "", err
}

// Reconcile every current peer in the same transaction; saved references remain immutable.
func (s *Store) removeOfflinePeers(ctx context.Context, tx *sql.Tx, o *equipment.PPObservation) error {
	readings, err := loadConcentratorReadings(ctx, tx, o.DeviceID)
	if err != nil {
		return err
	}
	seen := map[string]string{}
	for _, r := range o.Rows {
		if r.State == "ONLINE" {
			seen[r.Login] = o.Provenance.Ended
		}
	}
	for _, peer := range readings {
		if !peer.current {
			continue
		}
		// ignored: Only the status or membership is needed in this check.
		old, _, code := loadPreviousPPObservation(ctx, tx, peer.id)
		if code != equipment.OK {
			return errPresenceUnavailable
		}
		count := len(old.Rows)
		equipment.DropOfflineSeenElsewhere(old, seen)
		if len(old.Rows) == count {
			continue
		}
		old.ID = uuid.NewString()
		if !ppWithinBudget(old) {
			return errPresenceUnavailable
		}
		if code = s.insertPPObservationAndRows(ctx, tx, old); code != equipment.OK {
			return errPresenceUnavailable
		}
	}
	return ppGC(ctx, tx)
}

func applySessionHistoryPlan(ctx context.Context, tx *sql.Tx, o *equipment.PPObservation, plan pppoe.HistoryPlan) error {
	at, err := time.Parse(time.RFC3339Nano, o.Provenance.Ended)
	if err != nil {
		return err
	}
	for id, key := range plan.Rebind {
		// ignored: Only the SQL error is consumed; the result carries no required domain value.
		// ignored: Only the SQL error is needed; no result value is consumed.
		if _, err = tx.ExecContext(ctx, "UPDATE session_history SET session_key=? WHERE id=?", key, id); err != nil {
			return err
		}
	}
	for _, r := range plan.Connect {
		key := pppoe.SessionKey(r)
		body, err := json.Marshal(r)
		if err != nil {
			return err
		}
		// ignored: Only the SQL error is consumed; the result carries no required domain value.
		// ignored: Only the SQL error is needed; no result value is consumed.
		_, err = tx.ExecContext(ctx, "INSERT INTO session_history(device_id,login,session_key,kind,connected_at,row_js"+
			"on) VALUES(?,?,?,'session',?,?)",
			o.DeviceID, r.Login, key, pppoe.ConnectedAt(r, at), string(body))
		if err != nil {
			return err
		}
	}
	for _, id := range plan.Disconnect {
		// ignored: Only the SQL error is needed; no result value is consumed.
		if _, err = tx.ExecContext(ctx,
			"UPDATE session_history SET disconnected_at=?,reason=? WHERE id=?",
			equipment.PPTime(at),
			pppoe.UnknownDisconnectReason,
			id); err != nil {
			return err
		}
	}
	return nil
}
