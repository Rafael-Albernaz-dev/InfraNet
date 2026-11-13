package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/inventory"
)

func (s *Store) GetTablePreference(ctx context.Context, company, table string) (*inventory.TablePreference, inventory.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != company {
		return nil, inventory.Changed
	}
	if s.checkActive() != nil {
		return nil, inventory.Unavailable
	}
	if !inventory.ValidTablePreference(table, inventory.TablePreference{PageSize: 50}) {
		return nil, inventory.Invalid
	}
	var body string
	err := s.unit.QueryRowContext(ctx, "SELECT body FROM table_preferences WHERE table_id=?", table).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return s.initialTablePreference(ctx, table)
	}
	if err != nil {
		return nil, inventory.Unavailable
	}
	var p inventory.TablePreference
	if len(body) > 32*1024 || json.Unmarshal([]byte(body), &p) != nil || !inventory.ValidTablePreference(table, p) {
		return nil, inventory.Unavailable
	}
	return &p, inventory.OK
}
func (s *Store) initialTablePreference(ctx context.Context, table string) (*inventory.TablePreference, inventory.Code) {
	if table != "equipment" {
		return nil, inventory.OK
	}
	account := access.Account{}
	if loadInventoryColumns(ctx, s.db, &account) != nil {
		return nil, inventory.Unavailable
	}
	if len(account.Columns) == 0 {
		return nil, inventory.OK
	}
	for i, column := range account.Columns {
		if column == "block" {
			account.Columns[i] = "group"
		}
	}
	p := inventory.TablePreference{Columns: account.Columns, PageSize: 50}
	if c := s.writeTablePreference(ctx, table, p); c != inventory.OK {
		return nil, c
	}
	return &p, inventory.OK
}
func (s *Store) SetTablePreference(ctx context.Context, company, table string, p inventory.TablePreference) inventory.Code {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != company {
		return inventory.Changed
	}
	if s.checkActive() != nil {
		return inventory.Unavailable
	}
	return s.writeTablePreference(ctx, table, p)
}
func (s *Store) writeTablePreference(ctx context.Context, table string, p inventory.TablePreference) inventory.Code {
	if !inventory.ValidTablePreference(table, p) {
		return inventory.Invalid
	}
	body, err := json.Marshal(p)
	if err != nil || len(body) > 32*1024 {
		return inventory.Invalid
	}
	tx, err := s.unit.BeginTx(ctx, nil)
	if err != nil {
		return inventory.Unavailable
	}
	defer tx.Rollback()
	var count, exists int
	if tx.QueryRowContext(ctx,
		"SELECT count(*),count(CASE WHEN table_id=? THEN 1 END) FROM table_preferences",
		table).Scan(&count,
		&exists) != nil {
		return inventory.Unavailable
	}
	if count >= 64 && exists == 0 {
		return inventory.Limit
	}
	// ignored: SQLite result carries no preference revision; the transaction outcome is authoritative.
	_, err = tx.ExecContext(ctx, `INSERT INTO table_preferences(table_id,body) VALUES(?,?)
 ON CONFLICT(table_id) DO UPDATE SET body=excluded.body`, table, string(body))
	if err != nil || tx.Commit() != nil {
		return inventory.Unavailable
	}
	return inventory.OK
}
