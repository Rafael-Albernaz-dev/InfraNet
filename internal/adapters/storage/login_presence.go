package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pppoe"
	"time"
)

func loadCPELogins(ctx context.Context, q queryer, st *equipment.State) error {
	const query = `SELECT l.device_id,l.login,l.origin,l.created_at,count(r.row_id),
 COALESCE(max(r.row_json),''),COALESCE(json_extract(max(o.meta_json),'$.provenance.ended_at'),''),
 COALESCE((SELECT h.connected_at FROM session_history h WHERE h.device_id=k.concentrator_id
 AND h.login=l.login AND h.kind='session' ORDER BY h.connected_at DESC,h.id DESC LIMIT 1),''),
 COALESCE(p.last_online_at,'')
 FROM cpe_logins l LEFT JOIN radio_links k ON k.login=l.login AND json_extract(k.body,'$.cpe_id')=l.device_id
 LEFT JOIN pppoe_current c ON c.device_id=k.concentrator_id AND c.scope=k.scope
 LEFT JOIN pppoe_observations o ON o.id=c.observation_id
 LEFT JOIN pppoe_rows r ON r.observation_id=c.observation_id AND r.login=l.login
 LEFT JOIN device_presence p ON p.device_id=l.device_id GROUP BY l.device_id`
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	by := map[string]*equipment.Device{}
	for i := range st.Devices {
		by[st.Devices[i].ID] = &st.Devices[i]
	}
	for rows.Next() {
		var id, body string
		var count int
		login := &equipment.CPELogin{Session: &equipment.CPESessionPresence{}}
		presence := login.Session
		if err = rows.Scan(&id, &login.Login, &login.Origin, &login.CreatedAt, &count,
			&body, &presence.ReadAt, &presence.Since, &presence.LastOnline); err != nil {
			return err
		}
		if count == 1 {
			var row equipment.PPRow
			if err = json.Unmarshal([]byte(body), &row); err != nil {
				return err
			}
			presence.State = row.State
			if presence.State == "" {
				presence.State = "ONLINE"
			}
			if presence.State == "ONLINE" {
				presence.LastOnline = presence.ReadAt
			}
			if presence.Since == "" && presence.State == "ONLINE" {
				if at, e := time.Parse(time.RFC3339Nano, presence.ReadAt); e == nil {
					presence.Since = pppoe.ConnectedAt(row, at)
				}
			}
		}
		if d := by[id]; d != nil {
			d.CPELogin = login
		}
	}
	return rows.Err()
}

// Preserve the last ONLINE collection before the immutable observation is GC'd.
// This shares the existing presence checkpoint, without sending a probe or
// creating device ping events for a session-derived state.
func checkpointEnlaceOnline(ctx context.Context, tx *sql.Tx, concentrator string) error {
	const query = `INSERT INTO device_presence(device_id,address,last_online_at,state,since,saved_at)
 SELECT d.id,d.address,json_extract(o.meta_json,'$.provenance.ended_at'),'','',
 CAST(strftime('%s',json_extract(o.meta_json,'$.provenance.ended_at')) AS INTEGER)
 FROM cpe_logins l JOIN devices d ON d.id=l.device_id
 JOIN radio_links k ON k.login=l.login AND json_extract(k.body,'$.cpe_id')=l.device_id
 JOIN pppoe_current c ON c.device_id=k.concentrator_id AND c.scope=k.scope
 JOIN pppoe_observations o ON o.id=c.observation_id
 JOIN pppoe_rows r ON r.observation_id=c.observation_id AND r.login=l.login
 WHERE c.device_id=? GROUP BY d.id HAVING count(r.row_id)=1
 AND COALESCE(json_extract(max(r.row_json),'$.state'),'ONLINE')='ONLINE'
 ON CONFLICT(device_id) DO UPDATE SET address=excluded.address,last_online_at=excluded.last_online_at,
 saved_at=excluded.saved_at`
	// ignored: the checkpoint has no domain result beyond successful persistence.
	_, err := tx.ExecContext(ctx, query, concentrator)
	return err
}
