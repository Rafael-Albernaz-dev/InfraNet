package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
)

func (s *Store) SavePPPCatalog(ctx context.Context, company string, device equipment.Device,
	credential uint64, result *equipment.PPReadResult) equipment.Code {
	if result == nil || result.Scope != equipment.PPPCatalogScope {
		return equipment.Invalid
	}
	entries := []equipment.PPPEntry{}
	for _, row := range result.Rows {
		if row.Entry == nil {
			return equipment.Invalid
		}
		entries = append(entries, *row.Entry)
	}
	body, err := json.Marshal(entries)
	if err != nil || len(body) > equipment.PPPCatalogBytes {
		return equipment.Limit
	}
	return s.ppWrite(ctx, company, func(tx *sql.Tx) equipment.Code {
		if code := s.verifyMonitoringSaveTarget(ctx, tx, company, device, credential); code != equipment.OK {
			return code
		}
		// ignored: Only the SQL error is needed; the catalog atomically replaces the previous reading.
		_, err := tx.ExecContext(ctx, `INSERT INTO ppp_catalog VALUES(?,?,?,?) ON CONFLICT(device_id) DO UPDATE SET
   at=excluded.at,limited=excluded.limited,body=excluded.body`, device.ID, result.Ended, result.CatalogLimited, string(body))
		if err != nil {
			return equipment.Unavailable
		}
		return equipment.OK
	})
}

func (s *Store) ReadPPPCatalog(
	ctx context.Context, company, device string,
) ([]equipment.PPPEntry, map[string]string, bool, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != company {
		return nil, nil, false, equipment.Changed
	}
	if s.checkActive() != nil {
		return nil, nil, false, equipment.Unavailable
	}
	rows, err := s.unit.QueryContext(ctx, "SELECT device_id,at,limited,body FROM ppp_catalog WHERE (?='' OR device_id=?)", device, device)
	if err != nil {
		return nil, nil, false, equipment.Unavailable
	}
	defer rows.Close()
	out := []equipment.PPPEntry{}
	dates := map[string]string{}
	limited := false
	for rows.Next() {
		var id, at, body string
		var capped bool
		var entries []equipment.PPPEntry
		if rows.Scan(&id, &at, &capped, &body) != nil || json.Unmarshal([]byte(body), &entries) != nil {
			return nil, nil, false, equipment.Unavailable
		}
		dates[id] = at
		limited = limited || capped
		for i := range entries {
			entries[i].DeviceID = id
		}
		out = append(out, entries...)
	}
	if rows.Err() != nil {
		return nil, nil, false, equipment.Unavailable
	}
	return out, dates, limited, equipment.OK
}

func (s *Store) PPPDropsToday(ctx context.Context, company, device, start string) (int, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != company {
		return 0, equipment.Changed
	}
	if s.checkActive() != nil {
		return 0, equipment.Unavailable
	}
	var n int
	err := s.unit.QueryRowContext(ctx, `SELECT count(*) FROM session_history WHERE device_id=? AND kind='session'
 AND disconnected_at>=? AND disconnected_at<>''`, device, start).Scan(&n)
	if err != nil {
		return 0, equipment.Unavailable
	}
	return n, equipment.OK
}

func (s *Store) PPPRadioLinks(ctx context.Context, company, device string) ([]equipment.RadioLink, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != company {
		return nil, equipment.Changed
	}
	if s.checkActive() != nil {
		return nil, equipment.Unavailable
	}
	rows, err := s.unit.QueryContext(ctx, "SELECT body FROM radio_links WHERE concentrator_id=? AND scope=?", device, equipment.PPScope)
	if err != nil {
		return nil, equipment.Unavailable
	}
	defer rows.Close()
	out := []equipment.RadioLink{}
	for rows.Next() {
		var body string
		var link equipment.RadioLink
		if rows.Scan(&body) != nil || json.Unmarshal([]byte(body), &link) != nil {
			return nil, equipment.Unavailable
		}
		out = append(out, link)
	}
	if rows.Err() != nil {
		return nil, equipment.Unavailable
	}
	return out, equipment.OK
}
