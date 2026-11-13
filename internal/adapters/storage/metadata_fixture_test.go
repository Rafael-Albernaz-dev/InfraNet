package storage

import (
	"github.com/google/uuid"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"reflect"
	"testing"
	"time"
)

type metadataFixture struct {
	t               *testing.T
	j               *journey
	service         *equipment.Service
	peer            *sshfixture.Server
	hold            chan struct{}
	credential, get equipment.Command
	current         *equipment.Device
	pending, change string
}

func newMetadataFixture(t *testing.T, change string) *metadataFixture {
	t.Helper()
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	s := newEquipment(t, j)
	hold := make(chan struct{}, 3)
	for range 3 {
		hold <- struct{}{}
	}
	peer, err := sshfixture.Start(sshfixture.Config{Hold: hold})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(peer.Close)

	cred, d := setupEquipment(t, j, s, peer.Port())
	query := trustEquipment(t, s, equipmentScope(j), d)
	started := equipmentCall(t, s, query, equipment.OK)
	done := awaitJob(t, s, query, started.Job.ID)
	if done.Result == nil || done.Code != equipment.OK {
		t.Fatal("initial reading missing")
	}
	get := equipmentScope(j)
	get.Op, get.ID = "inventory.get", d.ID
	current := equipmentCall(t, s, get, equipment.OK).Device
	var pending string
	if change == "name-pending" {
		query.Operation = uuid.NewString()
		pending = equipmentCall(t, s, query, equipment.OK).Job.ID
		deadline := time.Now().Add(time.Second)
		for peer.Commands.Load() < 4 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if peer.Commands.Load() != 4 {
			t.Fatal("pending query did not reach command barrier")
		}
		current = equipmentCall(t, s, get, equipment.OK).Device
		if current.LastAttempt == nil || current.LastAttempt.State != "STARTED" {
			t.Fatal("missing pending attempt")
		}
	}
	if change == "name-inactive" {
		toggle := equipmentScope(j)
		toggle.Op, toggle.ID, toggle.Operation, toggle.Revision, toggle.Active = "inventory.set-active", d.ID, uuid.NewString(),
			current.Revision, false
		current = equipmentCall(t, s, toggle, equipment.OK).Device
	}

	f := &metadataFixture{t: t, j: j, service: s, peer: peer, hold: hold, credential: cred, get: get, current: current,
		pending: pending, change: change}
	t.Cleanup(func() {
		if f.hold != nil {
			close(f.hold)
		}
	})
	return f
}
func (f *metadataFixture) editAndVerify() {
	t, j, s, cred, current, change := f.t, f.j, f.service, f.credential, f.current, f.change
	before := *current
	edit := equipmentScope(j)
	edit.Op, edit.ID, edit.Operation = "inventory.update", current.ID, uuid.NewString()
	edit.Revision, edit.Name, edit.Address, edit.Port, edit.Transport, edit.CredentialID = before.Revision, "Renamed router",
		before.Address, before.Port, before.Transport, before.CredentialID
	switch change {
	case "address":
		edit.Address = "127.0.0.2"
	case "port":
		edit.Port = 1
		if edit.Port == before.Port {
			edit.Port = 2
		}
	case "credential":
		cred.ID, cred.Operation, cred.Alias = uuid.NewString(), uuid.NewString(), "Other reader"
		equipmentCall(t, s, cred, equipment.OK)
		edit.CredentialID = cred.ID
	}

	f.verifyUpdate(edit)
}
func (f *metadataFixture) verifyUpdate(edit equipment.Command) {
	t, s, get, peer := f.t, f.service, f.get, f.peer
	before := *f.current
	auths, commands := peer.Auth.Load(), peer.Commands.Load()
	saved := equipmentCall(t, s, edit, equipment.OK).Device
	assertMetadataRevision(t, *saved, before, f.change)
	f.assertSavedReading(saved, before)
	f.assertEndpointTrust(saved, before)
	replay := equipmentCall(t, s, edit, equipment.OK).Device
	stored := equipmentCall(t, s, get, equipment.OK).Device
	if !reflect.DeepEqual(saved, replay) || !reflect.DeepEqual(saved, stored) || peer.Auth.Load() != auths ||
		peer.Commands.Load() != commands {
		t.Fatal("edit/replay did not persist coherently or connected")
	}

	if f.pending != "" {
		f.finishPending(saved, [2]int32{auths, commands})
	}
}
func (f *metadataFixture) finishPending(saved *equipment.Device, counts [2]int32) {
	t, j, s, get, peer, auths, commands := f.t, f.j, f.service, f.get, f.peer, counts[0], counts[1]
	status := equipmentScope(j)
	status.Op, status.JobID = "device.query-status", f.pending
	job := equipmentCall(t, s, status, equipment.OK).Job

	if job.State != "CANCELLED" || job.Code != equipment.Cancelled || job.Result != nil {
		t.Fatal("no-op save did not retain revision semantics", job)
	}
	close(f.hold)
	f.hold = nil
	s.Close()
	after := equipmentCall(t, s, get, equipment.OK).Device
	if !reflect.DeepEqual(after, saved) || after.LastAttempt.State != "STARTED" || peer.Auth.Load() != auths ||
		peer.Commands.Load() != commands {
		t.Fatal("cancelled stale query published or authenticated")
	}
}

func (f *metadataFixture) assertSavedReading(saved *equipment.Device, before equipment.Device) {
	t, change := f.t, f.change
	if change == "name-active" || change == "name-inactive" || change == "name-pending" {
		if !reflect.DeepEqual(saved.Result, before.Result) || !reflect.DeepEqual(saved.LastAttempt, before.LastAttempt) ||
			!reflect.DeepEqual(saved.TrustAcceptance, before.TrustAcceptance) {
			t.Fatal("metadata erased completed reading/attempt/trust")
		}
	} else if saved.Result != nil || saved.LastAttempt != nil {
		t.Fatal("changed connection retained incompatible reading")
	}

}
func (f *metadataFixture) assertEndpointTrust(saved *equipment.Device, before equipment.Device) {
	t, change := f.t, f.change
	if (change == "address" || change == "port") && (saved.Fingerprint != "" || saved.TrustAcceptance != nil) {
		t.Fatal("changed endpoint retained pin")
	}
	if change == "credential" && (saved.Fingerprint != before.Fingerprint || !reflect.DeepEqual(saved.TrustAcceptance,
		before.TrustAcceptance)) {
		t.Fatal("credential change erased unrelated endpoint trust")
	}

}
