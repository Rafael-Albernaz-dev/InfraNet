package storage

import (
	"context"
	"github.com/google/uuid"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"reflect"
	"testing"
)

func TestOperationalAutomaticFirstAcceptanceRollsBackAndRecovers(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	service := newEquipment(t, j)
	peer, err := sshfixture.Start(sshfixture.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	_, d := setupEquipment(t, j, service, peer.Port())
	q := equipmentScope(j)
	q.Op, q.ID, q.Revision, q.Operation = "trust.inspect", d.ID, 1, uuid.NewString()
	before, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK {
		t.Fatal(code)
	}
	if _, err := j.st.unit.Exec(`CREATE TRIGGER deny_acceptance BEFORE UPDATE OF trust_acceptance ON devices
 WHEN NEW.trust_acceptance IS NOT OLD.trust_acceptance BEGIN SELECT RAISE(ABORT,'isolated failure'); END`); err != nil {
		t.Fatal(err)
	}
	started := equipmentCall(t, service, q, equipment.OK)
	failed := awaitJob(t, service, q, started.Job.ID)
	if failed.Code != equipment.Code("PERSISTENCE_FAILED") {
		t.Fatal("pin persistence failure hidden", failed)
	}
	rolled, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || !reflect.DeepEqual(before, rolled) {
		t.Fatal("failed acceptance left partial pin/context/receipt")
	}
	if _, err := j.st.unit.Exec("DROP TRIGGER deny_acceptance"); err != nil {
		t.Fatal(err)
	}
	q.Operation = uuid.NewString()
	started = equipmentCall(t, service, q, equipment.OK)
	if job := awaitJob(t, service, q, started.Job.ID); job.Code != equipment.OK {
		t.Fatal(job)
	}
	get := equipmentScope(j)
	get.Op, get.ID = "inventory.get", d.ID
	accepted := equipmentCall(t, service, get, equipment.OK).Device
	acceptance := accepted.TrustAcceptance
	assertAutomaticAcceptance(t, accepted, peer.Fingerprint())
	assertRepeatedAutomaticPin(t, j, service, q, before, acceptance)
	if peer.Auth.Load() != 0 {
		t.Fatal("inspection authenticated")
	}
}
func assertAutomaticAcceptance(t *testing.T, accepted *equipment.Device, fingerprint string) {
	t.Helper()
	acceptance := accepted.TrustAcceptance
	if acceptance == nil || acceptance.Mode != "automatic-tofu" || acceptance.Company != companyX || acceptance.UID != 1000 ||
		acceptance.Address != "127.0.0.1" || acceptance.At == "" {
		t.Fatal("missing daemon acceptance context", acceptance)
	}
	if accepted.Fingerprint != fingerprint || accepted.TrustOrigin != "automatic-tofu" {
		t.Fatal("missing automatic pin")
	}
}
func assertRepeatedAutomaticPin(t *testing.T,
	j *journey,
	service *equipment.Service,
	q equipment.Command,
	before equipment.State,
	acceptance *equipment.TrustAcceptance) {
	t.Helper()
	acceptedRevision := uint64(2)
	q.Operation, q.Revision = uuid.NewString(), acceptedRevision
	started := equipmentCall(t, service, q, equipment.OK)
	if job := awaitJob(t, service, q, started.Job.ID); job.Code != equipment.OK {
		t.Fatal(job)
	}
	after, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK || len(after.Operations) != len(before.Operations)+1 ||
		!reflect.DeepEqual(after.Devices[0].TrustAcceptance, acceptance) {
		t.Fatal("same pin reaccepted or authenticated")
	}
}
