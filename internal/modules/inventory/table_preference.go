package inventory

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"unicode/utf8"
)

type TablePreference = equipment.TablePreference
type TableSort = equipment.TableSort

func validTableID(id string) bool {
	if len(id) < 1 || len(id) > 48 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

func ValidTablePreference(table string, p TablePreference) bool {
	if !validTableID(table) || !validPreferenceText(p.Search, 160) ||
		!uniquePreferenceValues(p.Columns, 32) || !uniquePreferenceValues(p.Manual, MaxDevices) || len(p.Filters) > 16 {
		return false
	}
	if p.PageSize != 25 && p.PageSize != 50 && p.PageSize != 100 && p.PageSize != 200 {
		return false
	}
	if p.Sort != nil && !validPreferenceKey(p.Sort.Column) {
		return false
	}
	for k, v := range p.Filters {
		if !validPreferenceKey(k) || !validPreferenceText(v, 200) {
			return false
		}
	}
	return true
}
func validPreferenceText(s string, limit int) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= limit
}
func validPreferenceKey(s string) bool { return s != "" && validPreferenceText(s, 200) }
func uniquePreferenceValues(values []string, limit int) bool {
	if len(values) > limit {
		return false
	}
	seen := map[string]bool{}
	for _, v := range values {
		if !validPreferenceKey(v) || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

type TablePreferenceStore interface {
	GetTablePreference(context.Context, string, string) (*TablePreference, Code)
	SetTablePreference(context.Context, string, string, TablePreference) Code
}

func (s *Service) tablePreference(ctx context.Context, q Command, v *View) Code {
	store, ok := s.Store.(TablePreferenceStore)
	if !ok {
		return Unavailable
	}
	if !validTableID(q.Table) {
		return Invalid
	}
	if q.Op == "inventory.table-preference-set" {
		if q.TablePreference == nil || !ValidTablePreference(q.Table, *q.TablePreference) {
			return Invalid
		}
		if c := store.SetTablePreference(ctx, q.Scope.Profile, q.Table, *q.TablePreference); c != OK {
			return c
		}
	}
	var c Code
	v.TablePreference, c = store.GetTablePreference(ctx, q.Scope.Profile, q.Table)
	return c
}
