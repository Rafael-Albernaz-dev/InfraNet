package storage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/importfile"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"sync"
	"testing"
	"time"
)

type contactStore struct {
	mu          sync.Mutex
	state       equipment.State
	failWrites  bool
	beforeWrite func(*equipment.State)
}

func (s *contactStore) ReadEquipment(context.Context, string) (equipment.State, equipment.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, err := json.Marshal(s.state)
	if err != nil {
		return equipment.State{}, equipment.Unavailable
	}
	var state equipment.State
	if json.Unmarshal(body, &state) != nil {
		return state, equipment.Unavailable
	}
	return state, equipment.OK
}
func (s *contactStore) UpdateEquipment(ctx context.Context, profile string, fn func(*equipment.State) equipment.Code) equipment.Code {
	state, code := s.ReadEquipment(ctx, profile)
	if code != equipment.OK {
		return code
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// All mutations in these scenarios are deliberately serialized by the store.
	body, err := json.Marshal(s.state)
	if err != nil {
		return equipment.Unavailable
	}
	if json.Unmarshal(body, &state) != nil {
		return equipment.Unavailable
	}
	if s.failWrites {
		return equipment.Unavailable
	}
	if s.beforeWrite != nil {
		s.beforeWrite(&state)
	}
	if code := fn(&state); code != equipment.OK {
		return code
	}
	s.state = state
	return equipment.OK
}

type contactRemote struct {
	equipment.Remote
	mu                 sync.Mutex
	entered            chan struct{}
	release            chan struct{}
	active, max, calls int
	fail               equipment.Code
}

func (r *contactRemote) FirstContact(ctx context.Context,
	d equipment.Device,
	_ string,
	_ string,
	guard func() equipment.Code) (*equipment.ContactResult,
	equipment.Code) {
	if code := guard(); code != equipment.OK {
		return nil, code
	}
	r.mu.Lock()
	r.active++
	r.calls++
	r.max = max(r.max, r.active)
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.active--; r.mu.Unlock() }()
	if r.entered != nil {
		r.entered <- struct{}{}
	}
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return nil, equipment.Cancelled
		}
	}
	if code := guard(); code != equipment.OK {
		return nil, code
	}
	if r.fail != "" {
		return nil, r.fail
	}
	return &equipment.ContactResult{Fingerprint: "SHA256:fixture",
			Observation: &equipment.Observation{Identity: "CCR",
				Board:   "CCR2004",
				Version: "7.1",
				At:      equipment.PPTime(time.Now())}},
		equipment.OK
}
func contactFixture(t *testing.T, r *contactRemote) (*inventory.Service, *contactStore, equipment.Command) {
	t.Helper()
	credential := uuid.NewString()
	store := &contactStore{state: equipment.State{Company: "company",
		WrappedKey: []byte{1},
		Credentials: []equipment.SealedCredential{{Metadata: equipment.Credential{ID: credential,
			Revision: 1}}}}}
	service := inventory.New(contactAuth{}, store, contactVault{}, r, contactIDs{}, time.Now)
	service.Parser, service.Digester = importfile.Parser{}, importfile.Parser{}
	t.Cleanup(service.Close)
	q := equipment.Command{Scope: access.Command{Profile: "company",
		ExpectedProfile:    "company",
		Activation:         "active",
		ExpectedActivation: "active"},
		Op:           "inventory.register",
		ID:           uuid.NewString(),
		Operation:    uuid.NewString(),
		Address:      "192.0.2.1",
		Port:         22,
		Role:         equipment.RoleConcentrator,
		Transport:    equipment.TransportSSH,
		CredentialID: credential}
	return service, store, q
}
func contactCall(t *testing.T, s *inventory.Service, q equipment.Command, want equipment.Code) equipment.View {
	t.Helper()
	view, code := s.Handle(context.Background(), q)
	if code != want {
		t.Fatalf("%s: got %s want %s", q.Op, code, want)
	}
	return view
}
func TestRegistrationProbesBeforePersistingAndPinsReadIdentity(t *testing.T) {
	remote := &contactRemote{entered: make(chan struct{}, 1), release: make(chan struct{})}
	service, store, q := contactFixture(t, remote)
	started := contactCall(t, service, q, equipment.OK)
	<-remote.entered
	state, _ := store.ReadEquipment(context.Background(), "company")
	if len(state.Devices) != 0 {
		t.Fatal("saved before successful contact")
	}
	close(remote.release)
	service.WG.Wait()
	q.Op, q.JobID = "inventory.register-status", started.Job.ID
	result := contactCall(t, service, q, equipment.OK)
	if result.Job.State != "SUCCEEDED" ||
		result.Device == nil ||
		result.Device.Name != "CCR" ||
		result.Device.Fingerprint != "SHA256:fixture" {
		t.Fatal(result)
	}
	if result.Device.TrustAcceptance.Mode != "automatic-tofu" {
		t.Fatal("missing automatic acceptance")
	}
}
func TestFailedRegistrationConfirmationKeepsRefusalWithoutRetry(t *testing.T) {
	remote := &contactRemote{fail: equipment.RadioCodeRemoteAuthFailed}
	service, store, q := contactFixture(t, remote)
	started := contactCall(t, service, q, equipment.OK)
	service.WG.Wait()
	state, _ := store.ReadEquipment(context.Background(), "company")
	if len(state.Devices) != 0 {
		t.Fatal("failure was saved without confirmation")
	}
	q.Op, q.JobID = "inventory.register-confirm", started.Job.ID
	confirmed := contactCall(t, service, q, equipment.OK)
	device := confirmed.Device
	if device == nil || device.Fingerprint != "" || device.LastAttempt == nil || device.LastAttempt.CredentialRevision != 1 {
		t.Fatal(device)
	}
	if inventory.ContactDue(*device, 1, time.Now().Add(time.Hour)) {
		t.Fatal("refused password retried")
	}
	if !inventory.ContactDue(*device, 2, time.Now()) {
		t.Fatal("credential revision did not release refusal")
	}
	if remote.calls != 1 {
		t.Fatal("confirmation retried password", remote.calls)
	}
}
func TestCredentialRevisionChangedDuringContactCannotPublish(t *testing.T) {
	remote := &contactRemote{entered: make(chan struct{}, 1), release: make(chan struct{})}
	service, store, q := contactFixture(t, remote)
	contactCall(t, service, q, equipment.OK)
	<-remote.entered
	if code := store.UpdateEquipment(context.Background(), "company", func(state *equipment.State) equipment.Code {
		state.Credentials[0].Metadata.Revision++
		return equipment.OK
	}); code != equipment.OK {
		t.Fatal(code)
	}
	close(remote.release)
	service.WG.Wait()
	state, _ := store.ReadEquipment(context.Background(), "company")
	if len(state.Devices) != 0 {
		t.Fatal("stale credential published")
	}
}
func TestImportCancellationKeepsStartedRowsAndCancelsRestOfThirty(t *testing.T) {
	remote := &contactRemote{entered: make(chan struct{}, 2), release: make(chan struct{})}
	service, store, q := contactFixture(t, remote)
	q.Op = "inventory.import-confirm"
	q.Import = &equipment.ImportInput{}
	store.state.CredentialDefaults = map[string]string{string(equipment.RoleAP): q.CredentialID}
	for n := 1; n <= 30; n++ {
		q.Import.Rows = append(q.Import.Rows,
			equipment.ImportRow{Line: n,
				Address: fmt.Sprintf("192.0.2.%d",
					n),
				Port:  "22",
				Role:  "AP",
				Group: "Norte > Torre"})
	}
	contactCall(t, service, q, equipment.OK)
	<-remote.entered
	<-remote.entered
	q.Op = "inventory.import-status"
	running := contactCall(t, service, q, equipment.OK)
	if running.OperationState != "RUNNING" || len(running.ImportItems) != 30 {
		t.Fatal(running)
	}
	other := q
	other.Op = "inventory.import-confirm"
	other.Operation = uuid.NewString()
	contactCall(t, service, other, equipment.Code("BUSY"))
	q.Op = "inventory.import-cancel"
	contactCall(t, service, q, equipment.OK)
	close(remote.release)
	service.WG.Wait()
	q.Op = "inventory.import-status"
	done := contactCall(t, service, q, equipment.OK)
	if done.OperationState != "COMPLETED" || done.ImportReceipt == nil || done.ImportReceipt.Created != 2 || done.ImportReceipt.Ignored != 28 {
		t.Fatal(done.ImportReceipt, done.OperationState)
	}
	state, _ := store.ReadEquipment(context.Background(), "company")
	if len(state.Devices) != 2 || len(state.Groups) != 2 || remote.max != 2 {
		t.Fatal(len(state.Devices), state.Groups, remote.max)
	}
	for _, device := range state.Devices {
		if inventory.GroupPath(state.Groups, device.GroupID) != "Norte › Torre" {
			t.Fatal(device)
		}
	}
}
func TestRefusedAutomaticContactDoesNotRepeatWhenAttemptWriteFails(t *testing.T) {
	remote := &contactRemote{fail: equipment.RadioCodeRemoteAuthFailed}
	service, store, q := contactFixture(t, remote)
	d := equipment.Device{ID: q.ID, Address: q.Address, Port: 22, Transport: equipment.TransportSSH,
		CredentialID: q.CredentialID, Active: true, Revision: 1}
	store.state.Devices = []equipment.Device{d}
	store.failWrites = true
	if code := service.AutomaticContact(context.Background(), "company", d, func() bool { return true }); code != equipment.Unavailable {
		t.Fatal(code)
	}
	if code := service.AutomaticContact(context.Background(), "company", d, func() bool { return true }); code != equipment.Changed {
		t.Fatal(code)
	}
	if remote.calls != 1 {
		t.Fatal("refused password retried after write failure", remote.calls)
	}
	if service.AutomaticContactDue("company", d, 1, time.Now().Add(time.Hour)) {
		t.Fatal("refusal expired without revision")
	}
	if !service.AutomaticContactDue("company", d, 2, time.Now()) {
		t.Fatal("credential change did not release refusal")
	}
}
func TestContactPublicationRevalidatesCredentialInsideFinalTransaction(t *testing.T) {
	remote := &contactRemote{}
	service, store, q := contactFixture(t, remote)
	store.beforeWrite = func(state *equipment.State) { state.Credentials[0].Metadata.Revision++ }
	started := contactCall(t, service, q, equipment.OK)
	service.WG.Wait()
	q.Op, q.JobID = "inventory.register-status", started.Job.ID
	result := contactCall(t, service, q, equipment.OK)
	if result.Job.Code != equipment.Revision {
		t.Fatal("stale credential committed after final guard", result.Job)
	}
	state, code := store.ReadEquipment(context.Background(), "company")
	if code != equipment.OK || len(state.Devices) != 0 {
		t.Fatal("stale device persisted", code)
	}
}

type contactAuth struct{}

func (contactAuth) WithAuthorization(_ context.Context, _ uint32, _ access.Command, f func(uint64) equipment.Code) equipment.Code {
	return f(1)
}

type contactIDs struct{}

func (contactIDs) ID() (string, equipment.Code) { return uuid.NewString(), equipment.OK }

type contactVault struct{ equipment.Vault }

func (contactVault) Available() bool                   { return true }
func (contactVault) Ready(string, string, []byte) bool { return true }
func (contactVault) Open(string, string, string, uint64, []byte, []byte) (string, equipment.Code) {
	return "synthetic", equipment.OK
}

func TestAutomaticContactAbandonsOfflineOrChangedCompanyBeforeAuthentication(t *testing.T) {
	remote := &contactRemote{}
	service, store, q := contactFixture(t, remote)
	d := equipment.Device{ID: q.ID, Address: q.Address, Port: 22, Transport: equipment.TransportSSH,
		CredentialID: q.CredentialID, Active: true, Revision: 1}
	store.state.Devices = []equipment.Device{d}
	if code := service.AutomaticContact(context.Background(), "company", d, func() bool { return false }); code != equipment.Changed {
		t.Fatal(code)
	}
	if remote.calls != 0 || store.state.Devices[0].Fingerprint != "" || store.state.Devices[0].LastAttempt != nil {
		t.Fatal("ineligible contact authenticated or persisted")
	}
}

func TestRegistrationReplayDoesNotAuthenticateAgainAndRejectsChangedInput(t *testing.T) {
	remote := &contactRemote{}
	service, _, q := contactFixture(t, remote)
	first := contactCall(t, service, q, equipment.OK)
	service.WG.Wait()
	replay := contactCall(t, service, q, equipment.OK)
	if replay.Job == nil || replay.Job.ID != first.Job.ID || remote.calls != 1 {
		t.Fatal("registration receipt retried contact", replay.Job, remote.calls)
	}
	q.Port++
	contactCall(t, service, q, equipment.Revision)
	if remote.calls != 1 {
		t.Fatal("changed request authenticated")
	}
}
