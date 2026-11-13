package presentation

import (
	"context"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/storage"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/equipment"
	"testing"
)

type modalIDs struct{}

func (modalIDs) ID() (string, equipment.Code) { return uuid.NewString(), equipment.OK }

// No Remote is provided: validation must not probe; valid input is explicitly confirmed after the unavailable probe.
func TestValidationModalBindingRealDaemon(t *testing.T) {
	rig := newOperationalRig(t, nil)
	app, st, ctx, provider := rig.app, rig.st, rig.ctx, rig.provider
	selected := registerFixtureCompany(t, app)
	profile, activation := selected.Profile, selected.Activation
	base := DeviceInput{ID: uuid.NewString(), Operation: uuid.NewString(), Name: "Router", Address: "10.0.0.1", Port: 22,
		Transport: string(equipment.TransportSSH), User: "reader", Secret: "  synthetic password  "}
	validateRegistrationInputs(t, app, st, profile, activation, base)
	for _, address := range []string{"10.0.0.1", " 2001:0DB8:0:0:0:0:0:1 ", "::ffff:192.0.2.1"} {
		d := base
		d.ID = uuid.NewString()
		d.Operation = uuid.NewString()
		d.Address = address
		d.Name = "ignored\nlegacy name"
		d.User = " reader "
		r := app.SaveEquipment(profile, activation, d)
		if r.Code != "OK" {
			t.Fatalf("valid address rejected: %s", r.Code)
		}
		r = finishRegistration(t, app, profile, activation, r, true)
		if r.Code != "OK" || r.Data == nil || r.Data.Device == nil {
			t.Fatal("confirmed registration", r.Code)
		}
		if app.SaveEquipment(profile, activation, d).Code != "OK" {
			t.Fatal("retry failed")
		}
	}
	state, cod := st.ReadEquipment(ctx, profile)
	if cod != equipment.OK || len(state.Devices) != 3 || len(state.Credentials) != 3 || len(state.Operations) != 3 {
		t.Fatal("atomic/idempotent enrollment failed")
	}
	if state.Devices[1].Address != "2001:db8::1" || state.Devices[0].Name != state.Devices[0].Address ||
		state.Credentials[0].Metadata.User != "reader" {
		t.Fatal("identifier canonicalization failed")
	}
	assertRegistrationSecrets(t, provider, state, base.Secret)
}

func validateRegistrationInputs(t *testing.T, app *App, st *storage.Store, profile, activation string,
	base DeviceInput) {
	t.Helper()
	for _, tc := range []struct {
		name string
		edit func(*DeviceInput)
		want string
	}{
		{"dotless", func(d *DeviceInput) { d.Address = "10001" }, "DEVICE_ADDRESS_INVALID"},
		{"host-port", func(d *DeviceInput) { d.Address = "10.0.0.1:8291" }, "DEVICE_ADDRESS_INVALID"},
		{"url", func(d *DeviceInput) { d.Address = "ssh://10.0.0.1" }, "DEVICE_ADDRESS_INVALID"},
		{"user-at", func(d *DeviceInput) { d.Address = "reader@10.0.0.1" }, "DEVICE_ADDRESS_INVALID"},
		{"unspecified", func(d *DeviceInput) { d.Address = "::" }, "DEVICE_ADDRESS_INVALID"},
		{"multicast", func(d *DeviceInput) { d.Address = "ff02::1" }, "DEVICE_ADDRESS_INVALID"},
		{"user", func(d *DeviceInput) { d.User = "bad\nuser" }, "CREDENTIAL_USER_INVALID"},
		{"port", func(d *DeviceInput) { d.Port = 65536 }, "DEVICE_PORT_INVALID"},
		{"secret", func(d *DeviceInput) { d.Secret = "" }, "CREDENTIAL_SECRET_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := base
			tc.edit(&d)
			r := app.SaveEquipment(profile, activation, d)
			if r.Code != tc.want {
				t.Fatalf("got %s want %s", r.Code, tc.want)
			}
			state, c := st.ReadEquipment(context.Background(), profile)
			if c != equipment.OK || len(state.Devices)+len(state.Credentials)+len(state.Operations) != 0 || len(state.WrappedKey) != 0 {
				t.Fatal("validation left partial data")
			}
		})
	}
}

func assertRegistrationSecrets(t *testing.T, provider *vault.Vault, state equipment.State, secret string) {
	t.Helper()
	for _, c := range state.Credentials {
		plain, cod := provider.Open(state.Installation, state.Company, c.Metadata.ID, c.Metadata.Revision, c.Envelope, state.WrappedKey)
		if cod != equipment.OK || plain != secret {
			t.Fatal("password changed or envelope unreadable")
		}
	}
}
