package storage

import (
	"context"
	"database/sql"
	"infradesktop/internal/modules/equipment"
)

func loadEquipmentGroups(ctx context.Context, q queryer, state *equipment.State) error {
	rows, err := q.QueryContext(ctx, "SELECT id,parent_id,name FROM device_groups ORDER BY rowid")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var group equipment.Group
		if err := rows.Scan(&group.ID, &group.Parent, &group.Name); err != nil {
			return err
		}
		state.Groups = append(state.Groups, group)
	}
	return rows.Err()
}

func persistEquipmentGroups(ctx context.Context, tx *sql.Tx, old, current []equipment.Group) equipment.Code {
	previous, next := map[string]equipment.Group{}, map[string]bool{}
	for _, group := range old {
		previous[group.ID] = group
	}
	for _, group := range current {
		next[group.ID] = true
		if previous[group.ID] == group {
			continue
		}
		// ignored: affected count is not needed; the statement error is checked.
		if _, err := tx.ExecContext(ctx, `INSERT INTO device_groups(id,parent_id,name) VALUES(?,?,?)
 ON CONFLICT(id) DO UPDATE SET parent_id=excluded.parent_id,name=excluded.name`, group.ID, group.Parent, group.Name); err != nil {
			return equipment.Unavailable
		}
	}
	for _, group := range old {
		if !next[group.ID] {
			// ignored: affected count is not needed; the statement error is checked.
			if _, err := tx.ExecContext(ctx, "DELETE FROM device_groups WHERE id=?", group.ID); err != nil {
				return equipment.Unavailable
			}
		}
	}
	return equipment.OK
}
