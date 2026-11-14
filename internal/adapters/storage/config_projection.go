package storage

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func loadConfigSummaries(ctx context.Context, q queryer, st *equipment.State) error {
	rows, err := q.QueryContext(ctx, `SELECT v.device_id,max(v.created_at),COALESCE(s.unchanged,0)
 FROM config_versions v LEFT JOIN config_backup_status s ON s.device_id=v.device_id GROUP BY v.device_id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	by := map[string]*equipment.ConfigSummary{}
	for rows.Next() {
		var id string
		summary := &equipment.ConfigSummary{}
		if err = rows.Scan(&id, &summary.At, &summary.Unchanged); err != nil {
			return err
		}
		by[id] = summary
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for i := range st.Devices {
		st.Devices[i].Config = by[st.Devices[i].ID]
	}
	return nil
}
