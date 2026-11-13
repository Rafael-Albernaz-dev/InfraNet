package storage

import (
	"context"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"time"
	"unicode/utf8"
)

func (s *Store) PPSessionHistory(ctx context.Context, company string, in equipment.PPInput) (equipment.PPView, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := equipment.PPView{History: []equipment.PPSessionHistory{}}
	if s.unit == nil || s.unitID != company {
		return v, equipment.Changed
	}
	if s.checkActive() != nil {
		return v, equipment.Unavailable
	}
	if !validHistoryQuery(in) {
		return v, equipment.Invalid
	}
	generationQuery := "SELECT generation FROM session_history_generation WHERE singleton=1"
	if err := s.unit.QueryRowContext(ctx, generationQuery).Scan(&v.HistoryGeneration); err != nil {
		return v, equipment.Unavailable
	}
	if in.HistoryGeneration != "" && in.HistoryGeneration != v.HistoryGeneration {
		return v, equipment.CursorInvalid
	}
	cut := ""
	if in.HistoryDays > 0 {
		cut = equipment.PPTime(time.Now().Add(-time.Duration(in.HistoryDays) * 24 * time.Hour))
	}
	where := " FROM session_history WHERE device_id=? AND (login=? OR kind='gap') AND (disconn" +
		"ected_at='' OR disconnected_at>=?) AND (?='' OR reason=?)"
	args := []any{in.DeviceID, in.Login, cut, in.HistoryReason, in.HistoryReason}
	if err := s.unit.QueryRowContext(ctx, "SELECT count(*)"+where, args...).Scan(&v.Total); err != nil {
		return v, equipment.Unavailable
	}
	rows, err := s.unit.QueryContext(ctx, `SELECT id,kind,device_id,login,connected_at,disconnected_at,reason,row_json`+where+
		" ORDER BY connected_at DESC,id DESC LIMIT ? OFFSET ?", append(args, equipment.PageSize, in.Offset)...)
	if err != nil {
		return v, equipment.Unavailable
	}
	defer rows.Close()
	for rows.Next() {
		var row equipment.PPSessionHistory
		var raw string
		if rows.Scan(&row.ID,
			&row.Kind,
			&row.DeviceID,
			&row.Login,
			&row.Connected,
			&row.Disconnected,
			&row.Reason,
			&raw) != nil ||
			json.Unmarshal([]byte(raw),
				&row.Row) != nil {
			return v, equipment.Unavailable
		}
		v.History = append(v.History, row)
	}
	if rows.Err() != nil {
		return v, equipment.Unavailable
	}
	v.NextOffset = in.Offset + len(v.History)
	v.More = v.NextOffset < v.Total
	return v, equipment.OK
}

func validHistoryQuery(in equipment.PPInput) bool {
	if in.Login == "" ||
		len(in.Login) > 256 ||
		!utf8.ValidString(in.Login) ||
		len(in.HistoryReason) > 256 ||
		!utf8.ValidString(in.HistoryReason) ||
		in.Offset < 0 {
		return false
	}
	return in.HistoryDays == 0 || in.HistoryDays == 7 || in.HistoryDays == 30 || in.HistoryDays == 90
}
