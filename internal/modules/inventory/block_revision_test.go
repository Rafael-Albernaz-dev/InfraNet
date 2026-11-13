package inventory

import (
	"testing"

	"infradesktop/internal/modules/equipment"
)

func TestGroupOnlyKeepsRevisionAndStillRejectsStaleWrites(t *testing.T) {
	original := Device{ID: "00000000-0000-4000-8000-000000000001", Name: "Sintetico", Address: "192.0.2.1", Port: 22,
		Transport: equipment.TransportSSH, Role: equipment.RoleOther, Revision: 7, Active: true}
	for _, mode := range []string{"edit", "bulk"} {
		t.Run(mode, func(t *testing.T) {
			st := State{Devices: []Device{original}, Groups: []Group{{ID: "00000000-0000-4000-8000-000000000009", Name: "Torre sintetica"}}}
			if mode == "edit" {
				next := original
				next.GroupID = "00000000-0000-4000-8000-000000000009"
				if code := prepareDeviceUpdate(&st.Devices[0], Command{Op: "inventory.update", Revision: 7}, &next); code != OK {
					t.Fatal(code)
				}
				if next.Revision != 7 {
					t.Fatalf("block changed revision to %d", next.Revision)
				}
				if code := prepareDeviceUpdate(&st.Devices[0], Command{Op: "inventory.update", Revision: 6}, &next); code != Revision {
					t.Fatalf("stale accepted: %s", code)
				}
				return
			}
			group := "00000000-0000-4000-8000-000000000009"
			q := Command{Operation: "00000000-0000-4000-8000-000000000002", Bulk: &equipment.BulkUpdate{
				GroupID: &group, Items: []equipment.BulkItem{{ID: original.ID, Revision: 7}},
			}}
			var v View
			changed := false
			if code := applyBulk(&st, q, "synthetic", &v, &changed); code != OK {
				t.Fatal(code)
			}
			if st.Devices[0].Revision != 7 || v.BulkReceipt.Items[0].Revision != 7 {
				t.Fatalf("block changed revision: %d", st.Devices[0].Revision)
			}
			q.Operation = "00000000-0000-4000-8000-000000000003"
			q.Bulk.Items[0].Revision = 6
			if code := applyBulk(&st, q, "stale", &v, &changed); code != Revision {
				t.Fatalf("stale accepted: %s", code)
			}
		})
	}
}

func TestEndpointChangeStillAdvancesRevision(t *testing.T) {
	old := Device{ID: "00000000-0000-4000-8000-000000000004", Name: "Antes", Address: "192.0.2.1", Port: 22,
		Transport: equipment.TransportSSH, Role: equipment.RoleOther, Revision: 7, Active: true}
	next := old
	next.Name = "ignored"
	next.Port++
	next.GroupID = "00000000-0000-4000-8000-000000000009"
	if code := prepareDeviceUpdate(&old, Command{Op: "inventory.update", Revision: 7}, &next); code != OK || next.Revision != 8 {
		t.Fatalf("unexpected revision %d: %s", next.Revision, code)
	}
}
