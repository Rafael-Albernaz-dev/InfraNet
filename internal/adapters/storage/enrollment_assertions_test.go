package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"reflect"
	"testing"
)

func verifyEnrollmentReceiptAndConflicts(t *testing.T,
	j *journey,
	s *equipment.Service,
	q equipment.Command,
	st equipment.State) equipment.Command {
	t.Helper()
	cr := st.Credentials[0]
	v := vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil })
	if secret, c := v.Open(st.Installation, st.Company, cr.Metadata.ID, 1, cr.Envelope, st.WrappedKey); c != equipment.OK ||
		secret != q.Secret {
		t.Fatal("saved credential not usable")
	}
	b, _ := json.Marshal(st.Operations)
	if bytes.Contains(b, []byte(q.Secret)) {
		t.Fatal("secret in receipt")
	}
	// Lost response: a separate read reconciles by IDs, without a password.
	status := equipmentScope(j)
	status.Op = "inventory.enroll-status"
	status.ID = q.ID
	status.Operation = q.Operation
	got := equipmentCall(t, s, status, equipment.OK)
	b, _ = json.Marshal(got)
	if !bytes.Contains(b, []byte(`"operation_state":"COMMITTED"`)) ||
		got.Device == nil ||
		got.Device.ID != q.ID ||
		bytes.Contains(b, []byte(q.Secret)) {
		t.Fatal("receipt reconciliation failed")
	}
	status.Operation = uuid.NewString()
	got = equipmentCall(t, s, status, equipment.OK)
	b, _ = json.Marshal(got)
	if !bytes.Contains(b, []byte(`"operation_state":"NOT_FOUND"`)) {
		t.Fatal("missing receipt ambiguous")
	}
	conflict := q
	conflict.Secret = "different secret"
	equipmentCall(t, s, conflict, equipment.Revision)
	conflict = q
	conflict.Address = "192.0.2.2"
	equipmentCall(t, s, conflict, equipment.Revision)
	return status
}
func verifyEnrollmentEditIsolation(t *testing.T,
	j *journey,
	s *equipment.Service,
	q equipment.Command,
	st equipment.State,
	status equipment.Command) {
	t.Helper()
	// A normal edit has no password, preserves access, and does not replace a shared credential.
	edit := q
	edit.Op = "inventory.update"
	edit.Operation = uuid.NewString()
	edit.Revision = 1
	edit.CredentialID = st.Devices[0].CredentialID
	edit.User = ""
	edit.Secret = ""
	edit.Name = "Renamed"
	equipmentCall(t, s, edit, equipment.OK)
	final, _ := j.st.ReadEquipment(context.Background(), companyX)
	if !reflect.DeepEqual(st.Credentials, final.Credentials) || !bytes.Equal(st.WrappedKey, final.WrappedKey) {
		t.Fatal("edit changed stored access")
	}
	j.create(companyY, "Y", access.OK)
	j.selectCompany(companyY, access.OK)
	equipmentCall(t, s, q, equipment.Changed)
	equipmentCall(t, s, status, equipment.Changed)
	clean, _ := j.st.ReadEquipment(context.Background(), companyY)
	if len(clean.Devices)+len(clean.Credentials) != 0 {
		t.Fatal("late enrollment wrote wrong company")
	}
}
