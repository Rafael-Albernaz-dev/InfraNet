package storage

import (
	"context"
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"testing"
)

func TestPPPoEEquipmentWritesOnlyChangedRecords(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	if _, e := j.st.unit.Exec(
		"CREATE TRIGGER no_credential_rewrite BEFORE UPDATE ON credentials " +
			"BEGIN SELECT RAISE(ABORT,'unchanged credential rewritten'); END"); e != nil {
		t.Fatal(e)
	}
	before, c := j.st.ReadEquipment(ctx, companyX)
	if c != equipment.OK {
		t.Fatal(c)
	}
	if c = j.st.UpdateEquipment(ctx, companyX, func(st *equipment.State) equipment.Code {
		st.Devices[0].Identity = "changed"
		return equipment.OK
	}); c != equipment.OK {
		t.Fatal("rewrote unchanged credential", c)
	}
	after, c := j.st.ReadEquipment(ctx, companyX)
	a, _ := json.Marshal(before.Credentials)
	b, _ := json.Marshal(after.Credentials)
	if c != equipment.OK || string(a) != string(b) || after.Devices[0].ID != target.Device.ID || after.Devices[0].Identity != "changed" {
		t.Fatal("unexpected delta", c)
	}
}
