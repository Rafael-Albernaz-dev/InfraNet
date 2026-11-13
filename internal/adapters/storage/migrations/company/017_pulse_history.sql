CREATE TABLE device_presence (
 device_id TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE RESTRICT,
 address TEXT NOT NULL, last_online_at TEXT NOT NULL, state TEXT NOT NULL,
 since TEXT NOT NULL, saved_at INTEGER NOT NULL
);
CREATE TABLE device_pulse_events (
 id INTEGER PRIMARY KEY, device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
 kind TEXT NOT NULL, started_at TEXT NOT NULL, ended_at TEXT NOT NULL DEFAULT '',
 before_value TEXT NOT NULL DEFAULT '', after_value TEXT NOT NULL DEFAULT ''
);
CREATE INDEX device_pulse_events_retention ON device_pulse_events(started_at);
CREATE TABLE device_boot_checkpoint (
 device_id TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE RESTRICT, boot_at TEXT NOT NULL
);
CREATE TABLE device_identity_checkpoint (
 device_id TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE RESTRICT,
 firmware TEXT NOT NULL, identity TEXT NOT NULL, model TEXT NOT NULL, at TEXT NOT NULL
);
CREATE TABLE session_history (
 id INTEGER PRIMARY KEY, device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
 login TEXT NOT NULL, session_key TEXT NOT NULL, kind TEXT NOT NULL,
 connected_at TEXT NOT NULL, disconnected_at TEXT NOT NULL DEFAULT '',
 reason TEXT NOT NULL DEFAULT '', row_json TEXT NOT NULL
);
CREATE UNIQUE INDEX session_history_open ON session_history(device_id,session_key) WHERE disconnected_at='' AND kind='session';
CREATE INDEX session_history_lookup ON session_history(device_id,login,connected_at);
CREATE TABLE session_collection_checkpoint (
 device_id TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE RESTRICT,
 last_success TEXT NOT NULL, gap_start TEXT NOT NULL DEFAULT ''
);

-- Public row values hydrate fields already stored in indexed columns. Old JSON
-- and saved references retain the same projection; no historical row is rewritten.
CREATE VIEW pppoe_row_values AS
 SELECT r.observation_id,r.row_id,r.ordinal,r.login,r.address,r.caller_id,r.search_text,r.uptime_ns,
 json_set(r.row_json,'$.row_id',r.row_id,'$.login',r.login,'$.address',COALESCE(r.address,''),
 '$.caller_id',COALESCE(r.caller_id,''),'$.state',COALESCE(json_extract(r.row_json,'$.state'),'ONLINE'),
 '$.duplicates',COALESCE(json_extract(r.row_json,'$.duplicates'),1),'$.sample',
 CASE WHEN json_extract(r.row_json,'$.sample_shared')=1 THEN json_extract(o.meta_json,'$.provenance.sample')
 ELSE json_extract(r.row_json,'$.sample') END) AS row_json,r.row_bytes
 FROM pppoe_rows r JOIN pppoe_observations o ON o.id=r.observation_id;

CREATE TABLE session_history_generation (singleton INTEGER PRIMARY KEY CHECK(singleton=1),generation INTEGER NOT NULL);
INSERT INTO session_history_generation VALUES(1,0);
