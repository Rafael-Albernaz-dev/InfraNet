package storage

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func ppTableColumn(key string) string {
	switch key {
	case "row":
		return "r.ordinal"
	case "state":
		return ppSessionStateSQL
	case "address":
		return "r.address"
	case "caller_id":
		return "r.caller_id"
	case "uptime":
		return "CASE WHEN " + ppSessionStateSQL + "='OFFLINE' THEN NULL ELSE r.uptime_ns END"
	case "duplicates":
		return "CASE WHEN " + ppSessionStateSQL + "='OFFLINE' OR r.login='' THEN 0 ELSE " + ppOnlineDuplicateCount + " END"
	case "download", "upload":
		return "CASE WHEN " + ppSessionStateSQL + "='OFFLINE' THEN NULL ELSE json_extract(r.row_json,'$." + key + ".min') END"
	case "plan":
		return "CASE WHEN json_extract(r.row_json,'$.context.secret_state') IN ('active','disabled') " +
			"THEN json_extract(r.row_json,'$.context.profile') END"
	case "vlan", "service", "interface":
		return "json_extract(r.row_json,'$.context." + key + "')"
	case "session_id":
		return "json_extract(r.row_json,'$.session_id')"
	case "linked_device":
		return "COALESCE((SELECT d.name FROM radio_links l JOIN devices d ON d.id=json_extract(l.body,'$.cpe_id') " +
			"WHERE l.concentrator_id=(SELECT device_id FROM pppoe_observations o WHERE o.id=r.observation_id) " +
			"AND l.scope='pppoe-active-v1' AND l.login=r.login),'Sem vínculo')"
	default:
		return "r.login"
	}
}
func ppTableSearch() string {
	return "table_session_search(r.row_json) || ' ' || " + ppTableColumn("linked_device") +
		" || ' ' || CASE WHEN (" + ppTableColumn("duplicates") + ")>1 THEN 'Com' ELSE 'Sem' END"
}
func (s *Store) ppTableOptions(ctx context.Context, observation string, v *equipment.PPView) equipment.Code {
	v.Options = map[string][]string{}
	for _, key := range []string{"state", "plan", "vlan", "service"} {
		column := ppTableColumn(key)
		rows, err := s.unit.QueryContext(ctx, "SELECT DISTINCT CAST("+column+" AS TEXT) AS value FROM pppoe_row_values r "+
			"WHERE r.observation_id=? AND "+column+" IS NOT NULL AND CAST("+column+" AS TEXT)<>'' ORDER BY value COLLATE table_portuguese LIMIT 200",
			observation)
		if err != nil {
			return equipment.Unavailable
		}
		values := []string{}
		for rows.Next() {
			var value string
			if rows.Scan(&value) != nil {
				rows.Close()
				return equipment.Unavailable
			}
			values = append(values, value)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return equipment.Unavailable
		}
		v.Options[key] = values
	}
	return equipment.OK
}
