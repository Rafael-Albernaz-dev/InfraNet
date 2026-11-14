CREATE TABLE config_versions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
 created_at TEXT NOT NULL, origin TEXT NOT NULL CHECK(origin IN ('auto','manual')),
 format TEXT NOT NULL CHECK(format IN ('rsc','cfg')), size INTEGER NOT NULL CHECK(size>0 AND size<=4194304),
 stamp TEXT NOT NULL, added INTEGER NOT NULL, removed INTEGER NOT NULL, body BLOB NOT NULL
);
CREATE INDEX config_versions_device ON config_versions(device_id,created_at DESC);
CREATE TABLE config_backup_status (
 device_id TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
 attempted_at TEXT NOT NULL, code TEXT NOT NULL, unchanged INTEGER NOT NULL, auto_at TEXT NOT NULL
);
