package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
)

func (s *Store) InterfaceAPs(ctx context.Context, company string) ([]equipment.InterfaceAPs, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != company {
		return nil, equipment.Changed
	}
	if s.checkActive() != nil {
		return nil, equipment.Unavailable
	}
	rows, err := s.unit.QueryContext(ctx, `SELECT b.concentrator_id,b.interface,b.ap_id,COALESCE(NULLIF(d.identity,''),d.name),
 COALESCE((SELECT group_concat(json_extract(e.value,'$.name'),', ') FROM ppp_catalog c,json_each(c.body) e
 WHERE c.device_id=b.concentrator_id AND json_extract(e.value,'$.kind')='server'
 AND json_extract(e.value,'$.interface')=b.interface),'')
 FROM ppp_interface_aps b JOIN devices d ON d.id=b.concentrator_id ORDER BY 1,2,3`)
	if err != nil {
		return nil, equipment.Unavailable
	}
	defer rows.Close()
	out := []equipment.InterfaceAPs{}
	for rows.Next() {
		var conc, iface, ap, name, service string
		if rows.Scan(&conc, &iface, &ap, &name, &service) != nil {
			return nil, equipment.Unavailable
		}
		if len(out) == 0 || out[len(out)-1].Concentrator != conc || out[len(out)-1].Interface != iface {
			out = append(out, equipment.InterfaceAPs{Concentrator: conc, ConcentratorName: name, Service: service,
				Interface: iface, APs: []string{}})
		}
		out[len(out)-1].APs = append(out[len(out)-1].APs, ap)
	}
	if rows.Err() != nil {
		return nil, equipment.Unavailable
	}
	return out, equipment.OK
}

func (s *Store) SetInterfaceAPs(ctx context.Context, company string, binding equipment.InterfaceAPs) equipment.Code {
	return s.ppWrite(ctx, company, func(tx *sql.Tx) equipment.Code {
		var role equipment.Role
		var active bool
		if tx.QueryRowContext(ctx, "SELECT role,active FROM devices WHERE id=?", binding.Concentrator).Scan(&role, &active) != nil ||
			role != equipment.RoleConcentrator || !active {
			return equipment.Invalid
		}
		for _, ap := range binding.APs {
			if tx.QueryRowContext(ctx, ("SELECT COALESCE(r.role,d.role),d.active FROM devices d "+
				"LEFT JOIN device_radio_roles r ON r.device_id=d.id WHERE d.id=?"), ap).Scan(&role, &active) != nil ||
				role != equipment.RoleAP || !active {
				return equipment.Invalid
			}
		}
		// ignored: the SQL result has no domain value; errors are returned.
		if _, err := tx.ExecContext(ctx, "DELETE FROM ppp_interface_aps WHERE concentrator_id=? AND interface=?",
			binding.Concentrator, binding.Interface); err != nil {
			return equipment.Unavailable
		}
		for _, ap := range binding.APs {
			// ignored: the SQL result has no domain value; errors are returned.
			if _, err := tx.ExecContext(ctx, "INSERT INTO ppp_interface_aps VALUES(?,?,?) ON CONFLICT DO NOTHING",
				binding.Concentrator, binding.Interface, ap); err != nil {
				return equipment.Unavailable
			}
		}
		return equipment.OK
	})
}

func writeCPELogin(ctx context.Context, tx *sql.Tx, link equipment.RadioLink) equipment.Code {
	const query = "INSERT INTO cpe_logins VALUES(?,?,?,?) ON CONFLICT(device_id) DO UPDATE SET login=excluded.login"
	// ignored: the SQL result has no domain value; errors are returned.
	if _, err := tx.ExecContext(ctx, query, link.CPE, link.Login, cpeLoginOrigin(link), link.At); err != nil {
		return equipment.Unavailable
	}
	return equipment.OK
}
func (s *Store) EnlaceSessions(ctx context.Context, company string) ([]equipment.EnlaceSession, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unit == nil || s.unitID != company {
		return nil, equipment.Changed
	}
	if s.checkActive() != nil {
		return nil, equipment.Unavailable
	}
	const query = `SELECT c.device_id,c.observation_id,r.row_json FROM pppoe_current c
 JOIN pppoe_row_values r ON r.observation_id=c.observation_id
 JOIN devices d ON d.id=c.device_id WHERE d.active=1 AND c.scope=?`
	rows, err := s.unit.QueryContext(ctx, query, equipment.PPScope)
	if err != nil {
		return nil, equipment.Unavailable
	}
	defer rows.Close()
	out := []equipment.EnlaceSession{}
	for rows.Next() {
		var session equipment.EnlaceSession
		var body string
		if rows.Scan(&session.Concentrator, &session.Observation, &body) != nil || json.Unmarshal([]byte(body), &session.Row) != nil {
			return nil, equipment.Unavailable
		}
		out = append(out, session)
	}
	if rows.Err() != nil {
		return nil, equipment.Unavailable
	}
	return out, equipment.OK
}

func cpeLoginOrigin(link equipment.RadioLink) string {
	if link.NewCPE == nil {
		return "manual"
	}
	return link.Origin
}

func validateEnlaceSource(ctx context.Context, tx *sql.Tx, link equipment.RadioLink, row equipment.PPRow) equipment.Code {
	if row.Context == nil || row.Context.Association != "associated" {
		return equipment.ObservationIncompatible
	}
	var count int
	const binding = "SELECT count(*) FROM ppp_interface_aps WHERE concentrator_id=? AND interface=? AND ap_id=?"
	if tx.QueryRowContext(ctx, binding, link.Concentrator, link.PPPInterface, link.AP).Scan(&count) != nil {
		return equipment.Unavailable
	}
	if count != 1 {
		return equipment.Revision
	}
	const catalog = `SELECT count(*) FROM ppp_catalog c,json_each(c.body) e WHERE c.device_id=?
 AND json_extract(e.value,'$.kind')='server' AND json_extract(e.value,'$.service')=?
 AND json_extract(e.value,'$.interface')=? AND COALESCE(json_extract(e.value,'$.invalid'),0)=0`
	if tx.QueryRowContext(ctx, catalog, link.Concentrator, row.Context.Service, link.PPPInterface).Scan(&count) != nil {
		return equipment.Unavailable
	}
	if count == 0 {
		return equipment.Revision
	}
	const owner = `SELECT count(*) FROM radio_links WHERE json_extract(body,'$.cpe_id')=? AND (concentrator_id!=? OR login!=?)`
	if tx.QueryRowContext(ctx, owner, link.CPE, link.Concentrator, link.Login).Scan(&count) != nil {
		return equipment.Unavailable
	}
	if count != 0 {
		return equipment.Revision
	}
	return equipment.OK
}
