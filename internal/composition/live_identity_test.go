package composition

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"testing"
)

type identityFixtureStore struct {
	*liveTestStore
	commits int
}

func (s *identityFixtureStore) UpdateEquipment(ctx context.Context,
	profile string,
	fn func(*equipment.State) equipment.Code) equipment.Code {
	state, code := s.ReadEquipment(ctx, profile)
	if code != equipment.OK {
		return code
	}
	state.Devices = append([]equipment.Device(nil), state.Devices...)
	if code := fn(&state); code != equipment.OK {
		return code
	}
	s.devices = state.Devices
	s.commits++
	return equipment.OK
}

type identityFixtureSession struct{ identity string }

func (s *identityFixtureSession) Read(context.Context, pulse.ReadTarget) (pulse.Reading, pulse.ReadFault) {
	return pulse.Reading{Identity: s.identity}, pulse.ReadOK
}
func (*identityFixtureSession) Close() {}
func TestLiveIdentityPersistsOnlyChangesAndRejectsStaleCredential(t *testing.T) {
	d := equipment.Device{ID: "d", Identity: "Antes", Role: equipment.RoleConcentrator, Active: true,
		Transport: equipment.TransportSSH, CredentialID: "c", Fingerprint: "trusted", Revision: 3}
	inv, base := liveTestInventory(d)
	store := &identityFixtureStore{liveTestStore: base}
	inv.store = store
	source := &identityFixtureSession{identity: "Nova"}
	session := &identityReadSession{ReadSession: source, inventory: inv, profile: "p",
		target: equipment.PPTarget{Device: d, Credential: store.credentials[0]}}
	for range 2 {
		reading, fault := session.Read(context.Background(), pulse.ReadTarget{})
		if fault != pulse.ReadOK || reading.Identity != "Nova" {
			t.Fatal(reading, fault)
		}
	}
	if store.commits != 1 || store.devices[0].Identity != "Nova" {
		t.Fatal("identity rewritten on identical tick", store.commits)
	}
	// A manual refresh can change persisted identity without changing configuration revision.
	store.devices[0].Identity = "Manual"
	if reading, fault := session.Read(context.Background(), pulse.ReadTarget{}); fault != pulse.ReadOK || reading.Identity != "Nova" {
		t.Fatal(reading, fault)
	}
	if store.devices[0].Identity != "Nova" || store.commits != 2 {
		t.Fatal("session cache hid an intervening manual identity", store.commits)
	}
	source.identity = "Outra"
	store.credentials[0].Metadata.Revision++
	if _, fault := session.Read(context.Background(), pulse.ReadTarget{}); fault != pulse.ReadLost {
		t.Fatal(fault)
	}
	if store.commits != 2 || store.devices[0].Identity != "Nova" {
		t.Fatal("stale credential published identity")
	}
}
