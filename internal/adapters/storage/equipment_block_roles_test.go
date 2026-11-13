package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
)

func TestNewRolesAreSavedWithoutStartingRemoteReads(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	s := newEquipment(t, j)
	peer, err := sshfixture.Start(sshfixture.Config{Family: "v6"})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	_, base := setupEquipment(t, j, s, peer.Port())
	for _, role := range []string{"edge", "switch", "station", "power-monitor", "ups", "rectifier", "solar-controller", "battery-monitor"} {
		t.Run(role, func(t *testing.T) {
			q := base
			q.ID = uuid.NewString()
			q.Operation = uuid.NewString()
			q.Role = equipment.Role(role)
			created := equipmentCall(t, s, q, equipment.OK)
			if created.Device == nil || string(created.Device.Role) != role {
				t.Fatal("role lost on create")
			}
			get := equipmentScope(j)
			get.Op = "inventory.get"
			get.ID = q.ID
			if got := equipmentCall(t, s, get, equipment.OK).Device; got == nil || string(got.Role) != role {
				t.Fatal("role lost on read")
			}
			for _, op := range []string{"trust.inspect", "device.query-start"} {
				read := equipmentScope(j)
				read.Op = op
				read.ID = q.ID
				read.Revision = 1
				read.Operation = uuid.NewString()
				if _, c := s.Handle(context.Background(), read); c != equipment.RadioCodeUnsupported {
					t.Fatalf("%s got %s", op, c)
				}
			}
			if peer.Handshakes.Load() != 0 || peer.Auth.Load() != 0 {
				t.Fatal("unsupported role contacted the peer")
			}
		})
	}
}
func TestDeviceGroupIsValidatedPersistedAndIsolated(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.create(companyY, "Y", access.OK)
	j.selectCompany(companyX, access.OK)
	service := newEquipment(t, j)
	_, base := setupEquipment(t, j, service, 2222)
	first, second := uuid.NewString(), uuid.NewString()
	for id, name := range map[string]string{first: "Torre Norte", second: "Área Sul"} {
		q := equipmentScope(j)
		q.Op, q.ID, q.Operation, q.Name = "inventory.group-create", id, uuid.NewString(), name
		equipmentCall(t, service, q, equipment.OK)
	}
	q := base
	q.ID, q.Operation, q.GroupID = uuid.NewString(), uuid.NewString(), first
	created := equipmentCall(t, service, q, equipment.OK).Device
	if created == nil || created.GroupID != first {
		t.Fatal("created group missing")
	}
	q.Op, q.Operation, q.Revision, q.GroupID = "inventory.update", uuid.NewString(), 1, second
	saved := equipmentCall(t, service, q, equipment.OK).Device
	if saved.GroupID != second || saved.Revision != 1 {
		t.Fatal("group changed collection revision", saved)
	}
	checkGroupNameBoundaries(t, j, service, second)
	get := equipmentScope(j)
	get.Op, get.ID = "inventory.get", q.ID
	j.selectCompany(companyY, access.OK)
	equipmentCall(t, service, get, equipment.Changed)
	foreign := equipmentScope(j)
	foreign.Op, foreign.ID = "inventory.get", q.ID
	equipmentCall(t, service, foreign, equipment.Invalid)
	j.selectCompany(companyX, access.OK)
	get = equipmentScope(j)
	get.Op, get.ID = "inventory.get", q.ID
	restored := equipmentCall(t, service, get, equipment.OK).Device
	if restored.GroupID != second || restored.GroupPath != strings.Repeat("á", 80) {
		t.Fatal("company change lost group", restored)
	}
}
func checkGroupNameBoundaries(t *testing.T, j *journey, service *equipment.Service, id string) {
	t.Helper()
	q := equipmentScope(j)
	q.Op, q.ID = "inventory.group-rename", id
	q.Operation, q.Name = uuid.NewString(), strings.Repeat("á", 80)
	equipmentCall(t, service, q, equipment.OK)
	for _, name := range []string{"", strings.Repeat("a", 81), "bad\nname", string([]byte{0xff}), " Norte ", "A > B"} {
		q.Operation, q.Name = uuid.NewString(), name
		equipmentCall(t, service, q, equipment.Invalid)
	}
	deletion := equipmentScope(j)
	deletion.Op, deletion.ID, deletion.Operation = "inventory.group-delete", id, uuid.NewString()
	equipmentCall(t, service, deletion, equipment.Invalid)
}
