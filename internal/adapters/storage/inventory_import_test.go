package storage

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"infradesktop/internal/adapters/importfile"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"infradesktop/internal/testsupport/contactprobe"
)

func importFixture(t *testing.T) (*journey, *inventory.Service) {
	t.Helper()
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	key := vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil })
	s := inventory.New(j.svc, j.st, key, &contactprobe.Adapter{Remote: routerssh.Adapter{}}, equipmentIDs{}, time.Now)
	seedImportCredential(t, j, s)
	s.Parser, s.Digester = importfile.Parser{}, importfile.Parser{}
	t.Cleanup(s.Close)
	return j, s
}

func importCall(t *testing.T, s *inventory.Service, q equipment.Command, want equipment.Code) equipment.View {
	t.Helper()
	if q.Op == "inventory.import-confirm" && q.Import != nil {
		for i := range q.Import.Rows {
			row := &q.Import.Rows[i]
			if row.Credential == "" && (row.Role == "Concentrador" || row.Role == "concentrator") {
				row.Credential = "import-fixture"
			}
		}
	}
	v, code := s.Handle(context.Background(), q)
	if code != want {
		t.Fatalf("%s got %s want %s", q.Op, code, want)
	}
	if q.Op == "inventory.import-confirm" && code == equipment.OK && v.ImportReceipt == nil {
		s.WG.Wait()
		q.Op = "inventory.import-status"
		v, code = s.Handle(context.Background(), q)
		if code != equipment.OK {
			t.Fatal("queue status", code)
		}
	}
	return v
}

func TestImportPreviewConfirmReplayAndConflicts(t *testing.T) {
	j, s := importFixture(t)
	q := equipmentScope(j)
	q.Op = "inventory.import-preview"
	q.Import = &equipment.ImportInput{Filename: "fixture.csv", Data: []byte("Nome,IP,Porta,Papel\nRouter,192.0.2.1,22,Concentrador\n")}
	view := importCall(t, s, q, equipment.OK)
	if len(view.ImportItems) != 1 || view.ImportItems[0].Status != "ready" {
		t.Fatal(view.ImportItems)
	}
	row := view.ImportItems[0].Row
	q.Op, q.Operation = "inventory.import-confirm", uuid.NewString()
	q.Import = &equipment.ImportInput{Rows: []equipment.ImportRow{row}}
	first := importCall(t, s, q, equipment.OK)
	if first.ImportReceipt == nil || first.ImportReceipt.Created != 1 {
		t.Fatal(first.ImportReceipt)
	}
	replay := importCall(t, s, q, equipment.OK)
	if replay.ImportReceipt == nil || replay.ImportReceipt.Created != 1 {
		t.Fatal(replay.ImportReceipt)
	}
	q.Import.Rows[0].Address = "192.0.2.9"
	importCall(t, s, q, equipment.Revision)
	q.Import.Rows[0].Address = row.Address
	q.Operation = uuid.NewString()
	importCall(t, s, q, equipment.Revision)
	state, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || len(state.Devices) != 1 || len(state.Operations) != 2 {
		t.Fatal(code, len(state.Devices), len(state.Operations))
	}
}

func TestImportRejectsWholeBatchAfterConflict(t *testing.T) {
	j, s := importFixture(t)
	q := equipmentScope(j)
	q.Op, q.Operation = "inventory.import-confirm", uuid.NewString()
	row := equipment.ImportRow{Line: 2, Name: "Router", Address: "192.0.2.1", Port: "22", Role: "AP"}
	other := row
	other.Line = 3
	q.Import = &equipment.ImportInput{Rows: []equipment.ImportRow{row, other}}
	importCall(t, s, q, equipment.Revision)
	st, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || len(st.Devices) != 0 || len(st.Operations) != 0 {
		t.Fatal("partial write", code)
	}
}

func TestImportPersistsFiveHundredWithinBudget(t *testing.T) {
	j, s := importFixture(t)
	q := equipmentScope(j)
	q.Op, q.Operation = "inventory.import-confirm", uuid.NewString()
	q.Import = &equipment.ImportInput{}
	for i := 0; i < 500; i++ {
		address := fmt.Sprintf("192.0.2.%d", i)
		if i >= 250 {
			address = fmt.Sprintf("198.51.100.%d", i-250)
		}
		row := equipment.ImportRow{Line: i + 1, Name: fmt.Sprintf("Router %d", i), Address: address, Port: "22", Role: "AP"}
		q.Import.Rows = append(q.Import.Rows, row)
	}
	started := time.Now()
	initial, code := s.Handle(context.Background(), q)
	if code != equipment.OK || initial.OperationState != "RUNNING" {
		t.Fatal(code, initial.OperationState)
	}
	elapsed := time.Since(started)
	t.Logf("500 registros: %s; orçamento: %s", elapsed, equipment.PPPersistenceBudget)
	if elapsed >= equipment.PPPersistenceBudget {
		t.Fatal("queue admission budget", elapsed)
	}
	s.WG.Wait()
	q.Op = "inventory.import-status"
	got := importCall(t, s, q, equipment.OK)
	if got.ImportReceipt == nil || got.ImportReceipt.Created != 500 {
		t.Fatal("queue result", got.ImportReceipt)
	}
	state, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || len(state.Devices) != 500 {
		t.Fatal(code, len(state.Devices))
	}
}

func TestImportConfirmRejectsMoreThanFiveHundred(t *testing.T) {
	j, s := importFixture(t)
	q := equipmentScope(j)
	q.Op, q.Operation = "inventory.import-confirm", uuid.NewString()
	row := equipment.ImportRow{Line: 1, Name: "A", Address: "192.0.2.1", Port: "22", Role: "AP"}
	q.Import = &equipment.ImportInput{Rows: make([]equipment.ImportRow, 501)}
	for i := range q.Import.Rows {
		q.Import.Rows[i] = row
	}
	importCall(t, s, q, equipment.Invalid)
	st, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || len(st.Devices) != 0 {
		t.Fatal("excess rows wrote devices", code)
	}
}

func seedImportCredential(t *testing.T, j *journey, s *inventory.Service) {
	t.Helper()
	state, code := j.st.ReadEquipment(context.Background(), j.st.unitID)
	if code != equipment.OK {
		t.Fatal(code)
	}
	id := uuid.NewString()
	envelope, wrapped, code := s.Vault.Seal(state.Installation, state.Company, id, 1, "synthetic-only", state.WrappedKey)
	if code != equipment.OK {
		t.Fatal(code)
	}
	code = j.st.UpdateEquipment(context.Background(), state.Company, func(current *equipment.State) equipment.Code {
		current.WrappedKey = wrapped
		current.Credentials = append(current.Credentials, equipment.SealedCredential{
			Metadata: equipment.Credential{ID: id, Alias: "import-fixture", User: "fixture", Revision: 1}, Envelope: envelope})
		current.CredentialDefaults = map[string]string{string(equipment.RoleAP): id}
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
}
