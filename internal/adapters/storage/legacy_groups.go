package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

func migrateLegacyGroups(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT DISTINCT block FROM devices WHERE block<>'' ORDER BY block")
	if err != nil {
		return err
	}
	blocks := []string{}
	for rows.Next() {
		var block string
		if err := rows.Scan(&block); err != nil {
			rows.Close()
			return err
		}
		blocks = append(blocks, block)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	used := map[string]bool{}
	for index, block := range blocks {
		name := legacyGroupName(block, used)
		id := fmt.Sprintf("00000000-0016-4000-8000-%012x", index+1)
		// ignored: affected count is not needed; the statement error is checked.
		if _, err := tx.Exec("INSERT INTO device_groups(id,parent_id,name) VALUES(?,'',?)", id, name); err != nil {
			return err
		}
		// ignored: affected count is not needed; the statement error is checked.
		if _, err := tx.Exec("UPDATE devices SET group_id=? WHERE block=?", id, block); err != nil {
			return err
		}
	}
	return nil
}

// Legacy blocks accepted separators, surrounding whitespace and case twins.
// Preserve their memberships and original block bytes; give every distinct
// block a valid, deterministic group name without merging groups.
func legacyGroupName(block string, used map[string]bool) string {
	base := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '>' || r == '›' {
			return '-'
		}
		return r
	}, block))
	if base == "" {
		base = "Grupo"
	}
	runes := []rune(base)
	if len(runes) > 80 {
		base = strings.TrimSpace(string(runes[:80]))
	}
	name := base
	for n := 2; legacyGroupNameUsed(used, name); n++ {
		prefix := []rune(base)
		if len(prefix) > 70 {
			prefix = prefix[:70]
		}
		name = fmt.Sprintf("%s (%d)", strings.TrimSpace(string(prefix)), n)
	}
	used[name] = true
	return name
}

func legacyGroupNameUsed(used map[string]bool, name string) bool {
	for previous := range used {
		if strings.EqualFold(previous, name) {
			return true
		}
	}
	return false
}
