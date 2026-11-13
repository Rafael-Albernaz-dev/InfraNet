package ipc

import (
	"context"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/foundation"
	"infradesktop/internal/testsupport/contactprobe"
	"infradesktop/internal/testsupport/jobprobe"
	"testing"
)

func prepareImportProbe(t *testing.T, j *ppJourney) *contactprobe.Adapter {
	t.Helper()
	probe := &contactprobe.Adapter{Remote: routerssh.Adapter{}}
	if !jobprobe.SetContactRemote(j.svc, probe) {
		t.Fatal("contact fixture unavailable")
	}
	configureImportCredential(t, j.client)
	return probe
}
func configureImportCredential(t *testing.T, client *AccountClient) {
	t.Helper()
	id := uuid.NewString()
	q := Request{Op: "credential.put", ExpectedProfile: client.profile, ExpectedActivation: client.activation,
		Equipment: &EquipmentInput{ID: id, Operation: uuid.NewString(), Alias: "import-fixture", User: "fixture", Secret: "synthetic-only"}}
	if result := client.Call(context.Background(), q); result.Code != foundation.OK {
		t.Fatal("fixture credential", result.Code)
	}
	q.Op = "credential.default-set"
	q.Equipment = &EquipmentInput{ID: uuid.NewString(), Operation: uuid.NewString(), Role: equipment.RoleAP, CredentialID: id}
	if result := client.Call(context.Background(), q); result.Code != foundation.OK {
		t.Fatal("fixture default", result.Code)
	}
}
func awaitImportIPC(t *testing.T, j *ppJourney) Response {
	t.Helper()
	if !jobprobe.WaitImport(j.svc) {
		t.Fatal("import fixture unavailable")
	}
	result := j.client.Call(context.Background(), Request{Op: "inventory.import-status", ExpectedProfile: j.client.profile,
		ExpectedActivation: j.client.activation, Equipment: &EquipmentInput{}})
	if result.Code != foundation.OK || result.Data == nil || result.Data.Equipment == nil ||
		result.Data.Equipment.ImportReceipt == nil || result.Data.Equipment.OperationState != "COMPLETED" {
		t.Fatal("queue terminal result", result.Code)
	}
	return result
}
