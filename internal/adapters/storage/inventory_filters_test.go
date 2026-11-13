package storage

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"testing"
)

func TestInventoryFiltersCoverAllPagesAndKeepTotal(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	s := newEquipment(t, j)
	_, base := setupEquipment(t, j, s, 2222)
	if c := j.st.UpdateEquipment(context.Background(), companyX, func(st *equipment.State) equipment.Code {
		st.Groups = []equipment.Group{{ID: "11111111-1111-4111-8111-111111111111", Name: "Norte"},
			{ID: "22222222-2222-4222-8222-222222222222", Name: "Sul"}}
		for i := 0; i < 21; i++ {
			st.Devices = append(st.Devices,
				equipment.Device{ID: uuid.NewString(),
					Name: fmt.Sprintf("Torre %02d",
						i),
					Identity: fmt.Sprintf("Torre %02d",
						i),
					Address: "192.0.2.1", Port: 22, Transport: equipment.TransportSSH, CredentialID: base.CredentialID,
					Role: equipment.RoleAP, Active: true, Revision: 1, GroupID: "11111111-1111-4111-8111-111111111111",
					Result: &equipment.Observation{Board: "Modelo A"}})
		}
		st.Devices = append(st.Devices,
			equipment.Device{ID: uuid.NewString(),
				Name:     "Torre inativa",
				Identity: "Torre inativa",
				Address:  "192.0.2.2",
				Port:     22, Transport: equipment.TransportSSH, Role: equipment.RoleCPE, Revision: 1, GroupID: "22222222-2222-4222-8222-222222222222",
				Result: &equipment.Observation{Board: "Modelo B"}})
		st.Devices = append(st.Devices, equipment.Device{ID: uuid.NewString(), Name: "Torre CPE", Identity: "Torre CPE", Address: "192.0.2.3",
			Port:      22,
			Transport: equipment.TransportSSH,
			Role:      equipment.RoleCPE,
			Revision:  1,
			GroupID:   "22222222-2222-4222-8222-222222222222",
			Active:    true,
			Result:    &equipment.Observation{Board: "Modelo B"}})
		for _, d := range []equipment.Device{
			{Role: equipment.RoleCPE, GroupID: "11111111-1111-4111-8111-111111111111", Result: &equipment.Observation{Board: "Modelo A"}},
			{Role: equipment.RoleAP, GroupID: "22222222-2222-4222-8222-222222222222", Result: &equipment.Observation{Board: "Modelo A"}},
			{Role: equipment.RoleAP, GroupID: "11111111-1111-4111-8111-111111111111", Result: &equipment.Observation{Board: "Modelo B"}},
		} {
			d.ID, d.Name, d.Address = uuid.NewString(), "Torre extra", "192.0.2.4"
			d.Identity = "Torre extra"
			d.Port, d.Transport, d.Active, d.Revision = 22, equipment.TransportSSH, true, 1
			st.Devices = append(st.Devices, d)
		}
		return equipment.OK
	}); c != equipment.OK {
		t.Fatal(c)
	}
	q := equipmentScope(j)
	q.Op = "inventory.list"
	q.Search = "torre"
	q.Role = equipment.RoleAP
	q.GroupID = "11111111-1111-4111-8111-111111111111"
	q.Model = "Modelo A"
	first := equipmentCall(t, s, q, equipment.OK)
	if len(first.Devices) != 16 || first.Next != 16 || first.Total != 27 {
		t.Fatalf("first page: %+v", first)
	}
	q.Cursor = first.Next
	second := equipmentCall(t, s, q, equipment.OK)
	if len(second.Devices) != 5 || second.Next != 0 || second.Total != 27 {
		t.Fatalf("second page: %+v", second)
	}
	if len(first.Groups) != 2 || fmt.Sprint(first.Models) != "[Modelo A Modelo B]" {
		t.Fatalf("options %v %v", first.Blocks, first.Models)
	}
	verifyInactiveAndEmptyBlockFilters(t, s, q)
}
func verifyInactiveAndEmptyBlockFilters(t *testing.T, s *equipment.Service, q equipment.Command) {
	t.Helper()
	q.Cursor = 0
	q.Role = equipment.RoleCPE
	q.GroupID = "22222222-2222-4222-8222-222222222222"
	q.Model = "Modelo B"
	q.IncludeInactive = true
	if got := equipmentCall(t, s, q, equipment.OK); len(got.Devices) != 2 || got.Total != 27 {
		t.Fatalf("inactive filter: %+v", got)
	}
	q.Role, q.GroupID, q.Model, q.Search = "", "", "", ""
	q.GroupEmpty = true
	if got := equipmentCall(t, s, q, equipment.OK); len(got.Devices) != 1 || got.Devices[0].GroupID != "" {
		t.Fatal("empty-block filter did not select the device without a block")
	}
}
