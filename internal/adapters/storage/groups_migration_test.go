package storage

import (
	"context"
	"encoding/json"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"testing"
)

func TestGroupsMigrationPreservesLegacyMembershipIdentityAndTableArrangement(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	dropPulseFixtureSchema(t, j)
	// ignored: Only the SQL error is consumed; the result carries no required domain value.
	if _, err := j.st.unit.Exec(`DROP TABLE device_groups; ALTER TABLE devices DROP COLUMN group_id;
 ALTER TABLE devices DROP COLUMN identity; DELETE FROM schema_migrations WHERE version=16`); err != nil {
		t.Fatal(err)
	}
	blocks := seedLegacyGroups(t, j)
	if err := j.st.validateUnitIdentity(j.st.unit, companyX); err != nil {
		t.Fatal(err)
	}
	state, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || len(state.Groups) != len(blocks) || !equipment.ValidGroups(state.Groups) {
		t.Fatal(code, state.Groups)
	}
	assertLegacyMembership(t, state, blocks)
	assertMigratedArrangement(t, j)
	assertHistoricalFieldsPreserved(t, j)
}

func seedLegacyGroups(t *testing.T, j *journey) []string {
	t.Helper()
	blocks := []string{"Centro", "centro", " Norte › Torre ", "   "}
	for index, block := range blocks {
		id := []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222",
			"33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444"}[index]
		if _, err := j.st.unit.Exec(`INSERT INTO devices(id,name,address,port,transport,revision,active,fingerprint,trust_origin,
 role,block,observation)
 VALUES(?,'Nome antigo','192.0.2.1',22,'ssh',1,1,'','','other',?,'{"identity":"CCR"}')`, id, block); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := j.st.unit.Exec(`INSERT INTO table_preferences VALUES('equipment',
 '{"columns":["state","block","cpu"],"filters":{"block":"Centro","state":"ONLINE"},
 "sort":{"column":"block","desc":false},"page_size":50}')`); err != nil {
		t.Fatal(err)
	}
	return blocks
}

func assertLegacyMembership(t *testing.T, state equipment.State, blocks []string) {
	t.Helper()
	ids := map[string]bool{}
	for i, d := range state.Devices {
		if d.Block != blocks[i] || d.Name != "Nome antigo" || d.Identity != "CCR" || d.GroupID == "" || ids[d.GroupID] {
			t.Fatal(d)
		}
		ids[d.GroupID] = true
	}
}

func assertMigratedArrangement(t *testing.T, j *journey) {
	t.Helper()
	var body string
	if err := j.st.unit.QueryRow("SELECT body FROM table_preferences WHERE table_id='equipment'").Scan(&body); err != nil {
		t.Fatal(err)
	}
	var preference equipment.TablePreference
	if err := json.Unmarshal([]byte(body), &preference); err != nil {
		t.Fatal(err)
	}
	if preference.Columns[1] != "group" ||
		preference.Sort.Column != "group" || preference.Filters["block"] != "" || preference.Filters["state"] != "ONLINE" {
		t.Fatal(body)
	}
}

func assertHistoricalFieldsPreserved(t *testing.T, j *journey) {
	t.Helper()
	code := j.st.UpdateEquipment(context.Background(), companyX, func(current *equipment.State) equipment.Code {
		current.Devices[0].Identity = "Nova"
		current.Devices[0].Name = "Sobrescrito"
		current.Devices[0].Block = ""
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
	after, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || after.Devices[0].Name != "Nome antigo" || after.Devices[0].Block != "Centro" ||
		after.Devices[0].Identity != "Nova" {
		t.Fatal(code, after.Devices[0])
	}
}
