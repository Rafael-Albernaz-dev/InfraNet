CREATE TABLE ppp_interface_aps (
 concentrator_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
 interface TEXT NOT NULL,
 ap_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
 PRIMARY KEY(concentrator_id, interface, ap_id)
);
CREATE TABLE cpe_logins (
 device_id TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
 login TEXT NOT NULL,
 origin TEXT NOT NULL,
 created_at TEXT NOT NULL
);
