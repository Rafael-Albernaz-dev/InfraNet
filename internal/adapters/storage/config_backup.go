package storage

import (
	"context"
	"database/sql"
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
)

func (s *Store) ConfigVersions(ctx context.Context, profile, device string) ([]cfgbackup.Stored, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != profile {
		return nil, equipment.Changed
	}
	if s.checkActive() != nil {
		return nil, equipment.Unavailable
	}
	rows, err := s.unit.QueryContext(ctx, `SELECT id,device_id,created_at,origin,format,size,stamp,added,removed
 FROM config_versions WHERE device_id=? ORDER BY created_at DESC,rowid DESC LIMIT 30`, device)
	if err != nil {
		return nil, equipment.Unavailable
	}
	defer rows.Close()
	versions := []cfgbackup.Stored{}
	for rows.Next() {
		var v cfgbackup.Stored
		if rows.Scan(&v.ID, &v.DeviceID, &v.CreatedAt, &v.Origin, &v.Format, &v.Size, &v.Stamp, &v.Added, &v.Removed) != nil {
			return nil, equipment.Unavailable
		}
		v.First = v.Added < 0
		versions = append(versions, v)
	}
	if rows.Err() != nil {
		return nil, equipment.Unavailable
	}
	return versions, equipment.OK
}
func (s *Store) ConfigStatus(ctx context.Context, profile, device string) (cfgbackup.Status, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var status cfgbackup.Status
	if s.unit == nil || s.unitID != profile {
		return status, equipment.Changed
	}
	if s.checkActive() != nil {
		return status, equipment.Unavailable
	}
	err := s.unit.QueryRowContext(ctx, `SELECT attempted_at,code,unchanged,auto_at FROM config_backup_status WHERE device_id=?`, device).
		Scan(&status.At, &status.Code, &status.Unchanged, &status.AutoAt)
	if err != nil && err != sql.ErrNoRows {
		return status, equipment.Unavailable
	}
	return status, equipment.OK
}
func (s *Store) SaveConfig(ctx context.Context, profile string, device equipment.Device,
	v *cfgbackup.Stored, status cfgbackup.Status) equipment.Code {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != profile {
		return equipment.Changed
	}
	if s.checkActive() != nil {
		return equipment.Unavailable
	}
	tx, err := s.unit.BeginTx(ctx, nil)
	if err != nil {
		return equipment.Unavailable
	}
	defer tx.Rollback()
	if !configSourceCurrent(ctx, tx, device, status.CredentialRevision) {
		return equipment.Changed
	}
	if v != nil {
		// ignored: only the result code or validation outcome is needed here.
		_, err = tx.ExecContext(ctx, "INSERT INTO config_versions VALUES(?,?,?,?,?,?,?,?,?,?)", v.ID, device.ID, v.CreatedAt,
			v.Origin, v.Format, v.Size, v.Stamp, v.Added, v.Removed, v.Body)
		if err != nil {
			return equipment.Unavailable
		}
		// ignored: only the result code or validation outcome is needed here.
		_, err = tx.ExecContext(ctx, `DELETE FROM config_versions WHERE device_id=? AND id NOT IN
   (SELECT id FROM config_versions WHERE device_id=? ORDER BY created_at DESC,rowid DESC LIMIT 30)`, device.ID, device.ID)
		if err != nil {
			return equipment.Unavailable
		}
	}
	// ignored: only the result code or validation outcome is needed here.
	autoAt := ""
	if status.Automatic {
		autoAt = status.At
	}
	// ignored: Exec result contains no required domain value.
	_, err = tx.ExecContext(ctx, `INSERT INTO config_backup_status VALUES(?,?,?,?,?) ON CONFLICT(device_id)
 DO UPDATE SET attempted_at=excluded.attempted_at,code=excluded.code,unchanged=excluded.unchanged,
 auto_at=CASE WHEN excluded.auto_at<>'' THEN excluded.auto_at ELSE config_backup_status.auto_at END`,
		device.ID, status.At, status.Code, status.Unchanged, autoAt)
	if err != nil || tx.Commit() != nil {
		return equipment.Unavailable
	}
	return equipment.OK
}

func (s *Store) ConfigVersion(ctx context.Context, profile, device, id string) (cfgbackup.Stored, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var v cfgbackup.Stored
	if s.unit == nil || s.unitID != profile {
		return v, equipment.Changed
	}
	if s.checkActive() != nil {
		return v, equipment.Unavailable
	}
	err := s.unit.QueryRowContext(ctx, `SELECT id,body FROM config_versions WHERE device_id=? AND id=?`, device, id).Scan(&v.ID, &v.Body)
	if err != nil {
		return v, equipment.Unavailable
	}
	return v, equipment.OK
}

func configSourceCurrent(ctx context.Context, tx *sql.Tx, device equipment.Device, expectedCredential uint64) bool {
	var revision, credentialRevision uint64
	if tx.QueryRowContext(ctx, "SELECT revision FROM devices WHERE id=? AND active=1", device.ID).Scan(&revision) != nil ||
		revision != device.Revision {
		return false
	}
	if expectedCredential != 0 {
		if tx.QueryRowContext(ctx, "SELECT revision FROM credentials WHERE id=?", device.CredentialID).Scan(&credentialRevision) != nil ||
			credentialRevision != expectedCredential {
			return false
		}
	}
	return true
}
