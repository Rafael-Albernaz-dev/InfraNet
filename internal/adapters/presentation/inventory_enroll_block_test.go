package presentation

import (
	"infradesktop/internal/testsupport/sshfixture"
	"testing"

	"github.com/google/uuid"
	"infradesktop/internal/modules/equipment"
)

func TestRegistrationWithCredentialAndGroupThroughBinding(t *testing.T) {
	r := newOperationalRig(t)
	const password = "synthetic account password"
	if r.app.Register("technical", password).Code != "OK" || r.app.AckRecovery().Code != "OK" ||
		r.app.Login("technical", password, false).Code != "OK" {
		t.Fatal("synthetic account")
	}
	created := r.app.CreateCompany("Synthetic")
	selected := r.app.SelectCompany(created.Companies[0].ID)

	group := uuid.NewString()
	if result := r.app.CreateEquipmentGroup(selected.Profile, selected.Activation, uuid.NewString(), group, "", "Norte"); result.Code != "OK" {
		t.Fatal(result.Code)
	}
	peer, err := sshfixture.Start(sshfixture.Config{Family: "v7", ServerVersion: "SSH-2.0-ROSSSH"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { peer.Close() })
	input := DeviceInput{ID: uuid.NewString(), Operation: uuid.NewString(), Name: "Synthetic AP", GroupID: group,
		Role: string(equipment.RoleAP), Address: "127.0.0.1", Port: peer.Port(), Transport: string(equipment.TransportSSH),
		User: "reader", Secret: sshfixture.Password}
	result := r.app.SaveEquipment(selected.Profile, selected.Activation, input)
	result = finishRegistration(t, r.app, selected.Profile, selected.Activation, result, false)
	if result.Code != "OK" || result.Data == nil || result.Data.Device == nil || result.Data.Device.GroupID != group ||
		result.Data.Device.Identity == "" || result.Data.Device.Fingerprint != peer.Fingerprint() {
		t.Fatal("atomic enrollment lost block", result.Code)
	}

	assertInlineRegistrationReplay(t, r.app, selected.Profile, selected.Activation, input, peer)

	got := r.app.GetEquipment(selected.Profile, selected.Activation, input.ID)
	if got.Code != "OK" || got.Data.Device.GroupID != group || got.Data.Device.CredentialID == "" {
		t.Fatal("enrolled block or credential missing", got.Code)
	}
}

func assertInlineRegistrationReplay(t *testing.T, app *App, profile, activation string, input DeviceInput, peer *sshfixture.Server) {
	t.Helper()
	auths := peer.Auth.Load()
	replay := app.SaveEquipment(profile, activation, input)
	if replay.Code != "OK" || peer.Auth.Load() != auths {
		t.Fatal("inline replay authenticated", replay.Code)
	}
	changed := input
	changed.Secret = "different synthetic secret"
	if app.SaveEquipment(profile, activation, changed).Code != "REVISION_CONFLICT" || peer.Auth.Load() != auths {
		t.Fatal("changed replay accepted")
	}
}
