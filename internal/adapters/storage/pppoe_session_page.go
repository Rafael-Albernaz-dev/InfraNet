package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"strings"
)

const ppOnlineDuplicateCount = "(SELECT count(*) FROM pppoe_row_values d WHERE d.observation_id=r.observation_id " +
	"AND d.login=r.login AND COALESCE(json_extract(d.row_json,'$.state'),'ONLINE')='ONLINE')"
const ppSessionStateSQL = "COALESCE(json_extract(r.row_json,'$.state'),'ONLINE')"

func ppSessionWhere(observation string, q *equipment.PPQuery) (string, []any) {
	where := "r.observation_id=?"
	args := []any{observation}
	if q.Search != "" {
		where += " AND instr(table_lower(" + ppTableSearch() + "),?)>0"
		args = append(args, strings.ToLower(q.Search))
	}
	if q.Duplicates || q.Duplicity == "with" {
		where += " AND " + ppSessionStateSQL + "='ONLINE' AND r.login<>'' AND " + ppOnlineDuplicateCount + ">1"
	}
	if q.Duplicity == "without" {
		where += " AND (" + ppSessionStateSQL + "='OFFLINE' OR r.login='' OR " + ppOnlineDuplicateCount + "<=1)"
	}
	for _, f := range []struct{ value, column string }{
		{q.State, ppSessionStateSQL}, {q.Plan, ppTableColumn("plan")}, {q.VLAN, ppTableColumn("vlan")}, {q.Service, ppTableColumn("service")},
	} {
		if f.value != "" {
			where += " AND CAST(" + f.column + " AS TEXT)=?"
			args = append(args, f.value)
		}
	}
	return where, args
}

func ppSessionOrder(q *equipment.PPQuery) string {
	column := ppTableColumn(q.Sort)
	if q.Sort == "" && q.PageSize > 0 {
		column = "r.ordinal"
	}
	direction := " ASC"
	if q.Desc {
		direction = " DESC"
	}
	order := column
	if q.Sort != "download" && q.Sort != "upload" && q.Sort != "uptime" && q.Sort != "vlan" && q.Sort != "duplicates" && q.Sort != "row" {
		order += " COLLATE table_portuguese"
	}
	return "(" + column + " IS NULL OR CAST(" + column + " AS TEXT) IN ('','—')) ASC," + order + direction + ",r.row_id ASC"
}

func scanPPSessionRow(rows interface{ Scan(...any) error }) (equipment.PPRow, equipment.Code) {
	var b string
	var count int
	var linkedName string
	if rows.Scan(&b, &count, &linkedName) != nil {
		return equipment.PPRow{}, equipment.Unavailable
	}
	var r equipment.PPRow
	if json.Unmarshal([]byte(b), &r) != nil {
		return equipment.PPRow{}, equipment.Unavailable
	}
	r.LinkedDeviceName = linkedName
	if r.Login != "" && r.State != "OFFLINE" {
		r.Duplicates = count
	}
	if r.State == "" {
		r.State = "ONLINE"
	}
	return r, equipment.OK
}

func (s *Store) PPPage(ctx context.Context, company string, in equipment.PPInput) (equipment.PPView, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := equipment.PPView{Rows: []equipment.PPRow{}}
	if s.unit == nil || s.unitID != company {
		return v, equipment.Changed
	}
	if s.checkActive() != nil {
		return v, equipment.Unavailable
	}
	o, c := ppMeta(ctx, s.unit, in.DeviceID, in.ObservationID)
	if c != equipment.OK {
		return v, c
	}
	v.Current = o
	q, size, c := validPPSessionPage(in)
	if c != equipment.OK {
		return v, c
	}
	where, args := ppSessionWhere(in.ObservationID, q)
	if s.unit.QueryRowContext(ctx, ppTableStatement("SELECT count(*) FROM pppoe_row_values r WHERE "+where),
		append([]any{in.ObservationID}, args...)...).Scan(&v.Total) != nil {
		return v, equipment.Unavailable
	}
	if in.Offset > v.Total {
		return v, equipment.CursorInvalid
	}
	order := ppSessionOrder(q)
	if q.PageSize > 0 {
		if c := s.ppTableOptions(ctx, in.ObservationID, &v); c != equipment.OK {
			return v, c
		}
	}
	args = append(args, size, in.Offset)
	rows, e := s.unit.QueryContext(ctx, ppTableStatement("SELECT json_set(r.row_json,'$.client',COALESCE("+
		ppClientColumn+",''),'$.linked_ap_name',"+ppTableColumn("linked_ap")+"),"+ppOnlineDuplicateCount+","+
		ppTableColumn("linked_device")+" FROM pppoe_row_values r WHERE "+where+" ORDER BY "+order+" LIMIT ? OFFSET ?"),
		append([]any{in.ObservationID}, args...)...)
	if e != nil {
		return v, equipment.Unavailable
	}
	if c := readPPSessionRows(rows, &v, q); c != equipment.OK {
		return v, c
	}
	v.NextOffset = in.Offset + len(v.Rows)
	v.More = v.NextOffset < v.Total
	return v, equipment.OK
}

func validPPSessionPage(in equipment.PPInput) (*equipment.PPQuery, int, equipment.Code) {
	q := in.Query
	if q == nil {
		q = &equipment.PPQuery{}
	}
	if !equipment.PPValidQuery(q) || in.Offset < 0 || in.Offset > equipment.PPMaxRows {
		return q, 0, equipment.Invalid
	}
	size := q.PageSize
	if size == 0 {
		size = equipment.PageSize
	} // Existing callers keep their original cursor size.
	return q, size, equipment.OK
}
func readPPSessionRows(rows *sql.Rows, v *equipment.PPView, q *equipment.PPQuery) equipment.Code {
	defer rows.Close()
	for rows.Next() {
		r, code := scanPPSessionRow(rows)
		if code != equipment.OK {
			return code
		}
		if q.PageSize == 0 {
			r.LinkedDeviceName = ""
		} // Legacy responses retain their original row projection.
		v.Rows = append(v.Rows, r)
	}
	if rows.Err() != nil {
		return equipment.Unavailable
	}
	return equipment.OK
}
func ppTableStatement(statement string) string {
	const counts = "WITH duplicates AS MATERIALIZED (SELECT login,count(*) AS total FROM pppoe_row_values " +
		"WHERE observation_id=? AND COALESCE(json_extract(row_json,'$.state'),'ONLINE')='ONLINE' GROUP BY login) "
	return counts + strings.ReplaceAll(statement, ppOnlineDuplicateCount, "COALESCE((SELECT total FROM duplicates WHERE login=r.login),0)")
}
