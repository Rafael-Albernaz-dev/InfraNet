package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/security"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEquipmentPersistenceIsolationConflictAndRollback(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.create(companyY, "Y", access.OK)
	j.selectCompany(companyX, access.OK)
	s := newEquipment(t, j)
	credential, d := setupEquipment(t, j, s, 2222)
	d2 := verifySharedEquipmentCredential(t, j, s, credential, d)
	verifyEquipmentCompanies(t, j, s, credential, d, d2)
	verifyEquipmentRestore(t, j, s, credential)
}
func verifySharedEquipmentCredential(t *testing.T, j *journey, s *equipment.Service, credential, d equipment.Command) equipment.Command {
	t.Helper()
	equipmentCall(t, s, d, equipment.OK)
	changed := d
	changed.Address = "192.0.2.2"
	equipmentCall(t, s, changed, equipment.Revision)
	stale := d
	stale.Op = "inventory.update"
	stale.Operation = uuid.NewString()
	stale.Revision = 0
	equipmentCall(t, s, stale, equipment.Revision)
	d2 := d
	d2.ID = uuid.NewString()
	d2.Operation = uuid.NewString()
	equipmentCall(t, s, d2, equipment.OK)
	list := equipmentScope(j)
	list.Op = "credential.list"
	if v := equipmentCall(t, s, list, equipment.OK); len(v.Credentials) != 1 || v.Credentials[0].Links != 2 {
		t.Fatal("shared references wrong")
	}
	replace := credential
	replace.Operation = uuid.NewString()
	replace.Revision = 1
	replace.Secret = "replacement test only"
	replace.Affected = 1
	equipmentCall(t, s, replace, equipment.Revision)
	replace.Affected = 2
	// Failing SQL rolls back ciphertext, revision, ledger and the wrapped key.
	if _, e := j.st.unit.Exec(
		"CREATE TRIGGER reject_credential BEFORE UPDATE ON credentials BEGIN SELECT RAISE(ABORT,'test'); END",
	); e != nil {
		t.Fatal(e)
	}
	equipmentCall(t, s, replace, equipment.Unavailable)
	st, _ := j.st.ReadEquipment(context.Background(), companyX)
	if st.Credentials[0].Metadata.Revision != 1 {
		t.Fatal("failed commit changed revision")
	}
	j.st.unit.Exec("DROP TRIGGER reject_credential")
	equipmentCall(t, s, replace, equipment.OK)
	equipmentCall(t, s, replace, equipment.OK)
	return d2
}
func verifyEquipmentCompanies(t *testing.T, j *journey, s *equipment.Service, credential, d, d2 equipment.Command) {
	t.Helper()
	list := equipmentScope(j)
	off := equipmentScope(j)
	off.Op = "inventory.set-active"
	off.ID = d.ID
	off.Operation = uuid.NewString()
	off.Revision = 1
	off.Active = false
	equipmentCall(t, s, off, equipment.OK)
	list.Op = "inventory.list"
	if v := equipmentCall(t, s, list, equipment.OK); len(v.Devices) != 1 || v.Devices[0].ID != d2.ID {
		t.Fatal("deactivation affected unrelated device")
	}
	list.Inactive = true
	if len(equipmentCall(t, s, list, equipment.OK).Devices) != 1 {
		t.Fatal("inactive missing")
	}
	off.Operation = uuid.NewString()
	off.Revision = 2
	off.Active = true
	equipmentCall(t, s, off, equipment.OK)
	oldScope := list
	j.selectCompany(companyY, access.OK)
	equipmentCall(t, s, oldScope, equipment.Changed)
	y := equipmentScope(j)
	y.Op = "inventory.list"
	if len(equipmentCall(t, s, y, equipment.OK).Devices) != 0 {
		t.Fatal("X leaked into Y")
	}
	yc, yd := setupEquipment(t, j, s, 2222)
	if yc.ID == credential.ID || yd.ID == d.ID {
		t.Fatal("identity reused")
	}
	y.CredentialID = credential.ID
	y.ID = uuid.NewString()
	y.Operation = uuid.NewString()
	y.Op = "inventory.create"
	y.Name = "foreign"
	y.Address = "127.0.0.1"
	y.Port = 2222
	y.Transport = equipment.TransportSSH
	equipmentCall(t, s, y, equipment.MissingCredential)
	j.selectCompany(companyX, access.OK)
}
func verifyEquipmentRestore(t *testing.T, j *journey, s *equipment.Service, credential equipment.Command) {
	t.Helper()
	// Reopen a coherent closed database set and use the same isolated provider.
	s.Close()
	dir := j.st.dir
	j.st.Close()
	store, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	j.st = store
	j.svc = access.New(store, security.Crypto{}, time.Now, 1000, "restart")
	v, c := j.svc.Handle(context.Background(), 1000, access.Command{Op: "access.login", Owner: "owner", Name: "technical", Password: password})
	if c != access.OK {
		t.Fatal(c)
	}
	j.q = access.Command{Session: v.Session, Owner: "owner", Profile: v.Profile, Activation: v.Activation}
	s = newEquipment(t, j)
	q := equipmentScope(j)
	q.Op = "inventory.list"
	if len(equipmentCall(t, s, q, equipment.OK).Devices) != 2 {
		t.Fatal("restart lost devices")
	}
	st, c := store.ReadEquipment(context.Background(), companyX)
	if c != equipment.OK {
		t.Fatal(c)
	}
	provider := vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil })
	plain, vc := provider.Open(st.Installation, st.Company,
		credential.ID, 2, st.Credentials[0].Envelope, st.WrappedKey)
	if vc != equipment.OK || plain != "replacement test only" {
		t.Fatal("coherent restore cannot decrypt")
	}
	missingProvider := vault.New(func() ([]byte, error) { return nil, fmt.Errorf("missing") })
	if _, vc = missingProvider.Open(st.Installation, st.Company, credential.ID, 2,
		st.Credentials[0].Envelope, st.WrappedKey); vc != equipment.VaultUnavailable {
		t.Fatal("restore without key accepted")
	}
	assertEquipmentPlaintextAbsent(t, dir)
}
func assertEquipmentPlaintextAbsent(t *testing.T, dir string) {
	t.Helper()
	// Scan all company/global files and live WAL/SHM, not only logical rows.
	if e := filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if bytes.Contains(b, []byte(sshfixture.Password)) || bytes.Contains(b, []byte("replacement test only")) {
			t.Fatal("remote plaintext reached storage")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}
