package storage

import (
	"context"
	"github.com/google/uuid"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"reflect"
	"sync"
	"testing"
)

func TestEquipmentDirectEnrollmentAtomicRetry(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	s := newEquipment(t, j)
	q := equipmentScope(j)
	q.Op = "inventory.enroll"
	q.ID = uuid.NewString()
	q.Operation = uuid.NewString()
	q.Name = "Direct router"
	q.Address = "127.0.0.1"
	q.Port = 22
	q.Transport = "ssh"
	q.User = "reader"
	q.Secret = "isolated direct secret"
	// Fail after the credential INSERT, before device and receipt commit.
	if _, e := j.st.unit.Exec("CREATE TRIGGER reject_device BEFORE INSERT ON devices BEGIN SELECT RAISE(ABORT,'fixture'); END"); e != nil {
		t.Fatal(e)
	}
	before, c := j.st.ReadEquipment(context.Background(), companyX)
	if c != equipment.OK {
		t.Fatal(c)
	}
	equipmentCall(t, s, q, equipment.Unavailable)
	after, _ := j.st.ReadEquipment(context.Background(), companyX)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed enrollment left key, credential, device or receipt")
	}
	if _, e := j.st.unit.Exec("DROP TRIGGER reject_device"); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	codes := make(chan equipment.Code, 2)
	for range 2 {
		wg.Go(func() { _, c := s.Handle(context.Background(), q); codes <- c })
	}
	wg.Wait()
	close(codes)
	for c := range codes {
		if c != equipment.OK {
			t.Fatal("double submit", c)
		}
	}
	st, _ := j.st.ReadEquipment(context.Background(), companyX)
	if len(st.Devices) != 1 ||
		len(st.Credentials) != 1 ||
		len(st.Operations) != 1 ||
		st.Devices[0].CredentialID != st.Credentials[0].Metadata.ID {
		t.Fatal("enrollment not atomic/idempotent")
	}
	status := verifyEnrollmentReceiptAndConflicts(t, j, s, q, st)
	verifyEnrollmentEditIsolation(t, j, s, q, st, status)
}
func TestEquipmentExistingAccessEnrollmentStatus(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	s := newEquipment(t, j)
	_, d := setupEquipment(t, j, s, 2222)
	status := equipmentScope(j)
	status.Op = "inventory.enroll-status"
	status.ID = d.ID
	status.Operation = d.Operation
	result := equipmentCall(t, s, status, equipment.OK)
	if result.OperationState != "COMMITTED" || result.Device == nil || result.Device.CredentialID != d.CredentialID {
		t.Fatal("existing access creation not reconciled")
	}
	status.ID = uuid.NewString()
	equipmentCall(t, s, status, equipment.Revision)
	old := equipmentScope(j)
	old.Op = "inventory.enroll"
	old.ID = uuid.NewString()
	old.Operation = uuid.NewString()
	old.Name = "Late"
	old.Address = "127.0.0.1"
	old.Port = 22
	old.Transport = "ssh"
	old.User = "reader"
	old.Secret = "fixture only"
	logout := j.q
	logout.Op = "access.logout"
	j.call(logout, access.OK)
	equipmentCall(t, s, old, access.Unauthorized)
	st, _ := j.st.ReadEquipment(context.Background(), companyX)
	if len(st.Devices) != 1 || len(st.Credentials) != 1 || len(st.Operations) != 2 {
		t.Fatal("stale post-logout enrollment mutated storage")
	}
}
