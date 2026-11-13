package storage

import (
	"context"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/security"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"reflect"
	"testing"
	"time"
)

type activationFixture struct {
	t                   *testing.T
	j                   *journey
	service             *equipment.Service
	peer                *sshfixture.Server
	query, get          equipment.Command
	expected            equipment.Device
	legacyName, pending string
}

func TestEquipmentActivationPreservesObservation(t *testing.T) {
	for _, scenario := range []string{"completed", "query-in-flight"} {
		t.Run(scenario, func(t *testing.T) {
			f := newActivationFixture(t, scenario)
			f.toggleAndReplay()
			f.reopenAndChangeEndpoint()
		})
	}
}
func newActivationFixture(t *testing.T, scenario string) *activationFixture {
	t.Helper()
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	service := newEquipment(t, j)
	hold := make(chan struct{}, 3)
	for range 3 {
		hold <- struct{}{}
	}
	peer, err := sshfixture.Start(sshfixture.Config{Family: "v7", Hold: hold})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(peer.Close)
	t.Cleanup(func() { close(hold) })
	_, d := setupEquipment(t, j, service, peer.Port())
	query := trustEquipment(t, service, equipmentScope(j), d)
	first := equipmentCall(t, service, query, equipment.OK)
	job := awaitJob(t, service, query, first.Job.ID)
	requireCompleteObservation(t, job)
	get := equipmentScope(j)
	get.Op, get.ID = "inventory.get", d.ID
	expected := *equipmentCall(t, service, get, equipment.OK).Device
	baseline, code := j.st.ReadEquipment(context.Background(), companyX)
	if code != equipment.OK {
		t.Fatal(code)
	}
	if !reflect.DeepEqual(expected.Result, job.Result) {
		t.Fatal("initial reading was not persisted")
	}
	pending := ""
	if scenario == "query-in-flight" {
		query.Operation = uuid.NewString()
		pending = equipmentCall(t, service, query, equipment.OK).Job.ID
		deadline := time.Now().Add(time.Second)
		for peer.Commands.Load() < 4 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if peer.Commands.Load() != 4 {
			t.Fatal("second query did not reach controlled blocking point")
		}
	}
	expected.LastAttempt = equipmentCall(t, service, get, equipment.OK).Device.LastAttempt
	return &activationFixture{t: t, j: j, service: service, peer: peer, query: query, get: get, expected: expected,
		legacyName: baseline.Devices[0].Name, pending: pending}
}
func requireCompleteObservation(t *testing.T, job equipment.Job) {
	t.Helper()
	if job.State != "SUCCEEDED" || job.Code != equipment.OK || job.Result == nil || len(job.Result.Interfaces) == 0 || job.Result.At == "" {
		t.Fatal("initial complete reading missing")
	}
}
func (f *activationFixture) check(device *equipment.Device) {
	f.t.Helper()
	if !reflect.DeepEqual(device, &f.expected) {
		f.t.Fatalf("activation changed completed data: got %#v expected %#v", device, &f.expected)
	}
	state, code := f.j.st.ReadEquipment(context.Background(), companyX)
	raw := f.expected
	raw.Name = f.legacyName
	if code != equipment.OK || len(state.Devices) != 1 || !reflect.DeepEqual(state.Devices[0], raw) {
		f.t.Fatal("stored observation or linked data changed")
	}
}
func (f *activationFixture) toggleAndReplay() {
	t, j, s := f.t, f.j, f.service
	expected := &f.expected
	auths := f.peer.Auth.Load()
	toggle := equipmentScope(j)
	toggle.Op, toggle.ID = "inventory.set-active", expected.ID
	toggle.Operation, toggle.Revision = uuid.NewString(), expected.Revision
	expected.Active, expected.Revision = false, expected.Revision+1
	f.check(equipmentCall(t, s, toggle, equipment.OK).Device)
	if f.pending != "" {
		status := f.query
		status.Op, status.JobID = "device.query-status", f.pending
		job := equipmentCall(t, s, status, equipment.OK).Job
		if job.State != "CANCELLED" || job.Code != equipment.Cancelled || job.Result != nil {
			t.Fatal("deactivation did not cancel in-flight query")
		}
	}
	f.check(equipmentCall(t, s, toggle, equipment.OK).Device)
	stale := toggle
	stale.Operation = uuid.NewString()
	equipmentCall(t, s, stale, equipment.Revision)
	f.check(equipmentCall(t, s, f.get, equipment.OK).Device)
	query := f.query
	query.Operation, query.Revision = uuid.NewString(), expected.Revision
	equipmentCall(t, s, query, equipment.Inactive)
	list := equipmentScope(j)
	list.Op = "inventory.list"
	if len(equipmentCall(t, s, list, equipment.OK).Devices) != 0 {
		t.Fatal("inactive device remained in active list")
	}
	list.Inactive = true
	items := equipmentCall(t, s, list, equipment.OK).Devices
	summary := *expected.Result
	summary.Interfaces = nil
	if len(items) != 1 || !reflect.DeepEqual(items[0].Result, &summary) {
		t.Fatal("inactive list lost last reading summary")
	}
	toggle.Operation, toggle.Revision, toggle.Active = uuid.NewString(), expected.Revision, true
	expected.Active, expected.Revision = true, expected.Revision+1
	f.check(equipmentCall(t, s, toggle, equipment.OK).Device)
	if f.peer.Auth.Load() != auths {
		t.Fatal("activation or refused query authenticated")
	}
}
func (f *activationFixture) reopenAndChangeEndpoint() {
	t, j := f.t, f.j
	f.service.Close()
	f.check(equipmentCall(t, f.service, f.get, equipment.OK).Device)
	dir := j.st.dir
	j.st.Close()
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	j.st = reopened
	j.svc = access.New(j.st, security.Crypto{}, time.Now, 1000, "restart")
	view := j.call(access.Command{Op: "access.login", Owner: "owner", Name: "technical", Password: password}, access.OK)
	j.q = access.Command{Session: view.Session, Owner: "owner", Profile: view.Profile, Activation: view.Activation}
	f.service = newEquipment(t, j)
	f.get = equipmentScope(j)
	f.get.Op, f.get.ID = "inventory.get", f.expected.ID
	f.check(equipmentCall(t, f.service, f.get, equipment.OK).Device)
	edit := equipmentScope(j)
	edit.Op, edit.ID, edit.Operation = "inventory.update", f.expected.ID, uuid.NewString()
	edit.Revision, edit.Name = f.expected.Revision, f.expected.Name
	edit.Address, edit.Port, edit.Transport = "127.0.0.2", f.expected.Port, f.expected.Transport
	edit.CredentialID = f.expected.CredentialID
	f.expected.Address, f.expected.Revision = edit.Address, f.expected.Revision+1
	f.expected.Identity, f.expected.Name = "", edit.Address
	f.expected.Fingerprint, f.expected.TrustOrigin, f.expected.Result = "", "", nil
	f.expected.TrustAcceptance, f.expected.LastAttempt = nil, nil
	f.check(equipmentCall(t, f.service, edit, equipment.OK).Device)
}
