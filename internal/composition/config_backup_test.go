package composition

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"infradesktop/internal/adapters/ipc"
	"infradesktop/internal/adapters/presentation"
	"infradesktop/internal/adapters/security"
	"infradesktop/internal/adapters/storage"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/foundation"
	"infradesktop/internal/testsupport/sshfixture"
)

type configJourney struct {
	client                            *ipc.AccountClient
	profile, activation, device, root string
	text                              atomic.Value
}

func newConfigJourney(t *testing.T) *configJourney {
	t.Helper()
	f := &configJourney{root: t.TempDir(), device: uuid.NewString()}
	f.text.Store("/ppp secret\nadd password=senha-canario\n" + strings.Repeat("set disabled=no comment=synthetic interface=ether1\n", 2100))
	if err := os.Chmod(f.root, 0700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"runtime", "state", "client"} {
		if err := os.Mkdir(filepath.Join(f.root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	store, err := storage.Open(filepath.Join(f.root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	uid := uint32(os.Getuid())
	auth := access.New(store, security.Crypto{}, time.Now, uid, "config", store)
	server, err := ipc.ListenAccess(filepath.Join(f.root, "runtime"), uid, auth, foundation.Status{Instance: "config"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	provider := func() ([]byte, error) { return bytes.Repeat([]byte{71}, 32), nil }
	composeConfig(server, auth, store, provider)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	f.client = ipc.NewAccountClient(filepath.Join(f.root, "runtime"), filepath.Join(f.root, "client"), uid)
	app := presentation.New(f.client)
	if app.Register("technical", "synthetic configuration password").Code != "OK" || app.AckRecovery().Code != "OK" ||
		app.Login("technical", "synthetic configuration password", false).Code != "OK" {
		t.Fatal("account")
	}
	company := app.CreateCompany("Configuration fixture")
	if company.Code != "OK" {
		t.Fatal(company.Code)
	}
	selected := app.SelectCompany(company.Companies[0].ID)
	f.profile, f.activation = selected.Profile, selected.Activation
	peer, err := sshfixture.Start(sshfixture.Config{Command: func(command string) (string, bool) {
		return f.text.Load().(string), command == "/export"
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(peer.Close)
	state, code := store.ReadEquipment(ctx, f.profile)
	if code != equipment.OK {
		t.Fatal(code)
	}
	credentialID := uuid.NewString()
	envelope, wrapped, code := vault.New(provider).Seal(state.Installation, f.profile, credentialID, 1, sshfixture.Password, nil)
	if code != equipment.OK {
		t.Fatal(code)
	}
	code = store.UpdateEquipment(ctx, f.profile, func(state *equipment.State) equipment.Code {
		state.WrappedKey = wrapped
		state.Credentials = []equipment.SealedCredential{{
			Metadata: equipment.Credential{ID: credentialID, Alias: "Synthetic", User: "reader", Revision: 1}, Envelope: envelope}}
		state.Devices = []equipment.Device{{ID: f.device, Name: "Router", Identity: "Router", Address: "127.0.0.1", Port: peer.Port(),
			Transport: equipment.TransportSSH, CredentialID: credentialID, Revision: 1, Active: true, Fingerprint: peer.Fingerprint(),
			Role: equipment.RoleConcentrator, Result: &equipment.Observation{Family: equipment.RouterFamilyV6, Version: "6.49.10"}}}
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
	return f
}
func (f *configJourney) call(t *testing.T, op string, in cfgbackup.Input, want foundation.Code) ipc.Response {
	t.Helper()
	r := f.client.Call(context.Background(), ipc.Request{Op: op, ExpectedProfile: f.profile, ExpectedActivation: f.activation, Config: &in})
	if r.Code != want {
		t.Fatalf("%s: got %s want %s", op, r.Code, want)
	}
	return r
}
func (f *configJourney) save(t *testing.T) ipc.Response {
	t.Helper()
	in := cfgbackup.Input{DeviceID: f.device}
	r := f.call(t, "config.save", in, foundation.OK)
	deadline := time.Now().Add(3 * time.Second)
	for r.Data.Config.Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
		r = f.call(t, "config.list", in, foundation.OK)
	}
	if r.Data.Config.Busy || r.Data.Config.Status.Code != equipment.OK {
		t.Fatal("backup did not complete")
	}
	return r
}
func TestConfigConfirmedDownloadAcrossDaemon(t *testing.T) {
	f := newConfigJourney(t)
	f.save(t)
	f.text.Store("/ppp secret\nadd password=nova-canario\n" + strings.Repeat("set disabled=no comment=synthetic interface=ether1\n", 2100))
	r := f.save(t)
	if len(r.Data.Config.Versions) != 2 {
		t.Fatal("history")
	}
	in := cfgbackup.Input{DeviceID: f.device, VersionID: r.Data.Config.Versions[0].ID}
	comparison := f.call(t, "config.compare", in, foundation.OK)
	assertConfigIPCPrivate(t, r, comparison)
	if len(comparison.Data.Config.Change.Lines) != 2000 || !comparison.Data.Config.Change.Truncated {
		t.Fatal("bounded comparison was rejected or not truncated")
	}
	if comparison.Data.Config.Change.Added != 1 || comparison.Data.Config.Change.Removed != 1 ||
		!strings.Contains(comparison.Data.Config.Change.Lines[1].Text, "********") {
		t.Fatal("masked comparison")
	}
	in.Path = filepath.Join(f.root, "configuracao.rsc")
	f.call(t, "config.download", in, foundation.Invalid)
	if _, err := os.Lstat(in.Path); !os.IsNotExist(err) {
		t.Fatal("download before confirmation")
	}
	in.Confirmed = true
	downloaded := f.call(t, "config.download", in, foundation.OK)
	body, err := os.ReadFile(in.Path)
	if err != nil || string(body) != f.text.Load().(string) {
		t.Fatal("complete file did not reach chosen destination")
	}
	info, err := os.Stat(in.Path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("private file mode")
	}
	safe, err := json.Marshal(downloaded)
	if err != nil || bytes.Contains(safe, []byte("canario")) {
		t.Fatal("download returned plaintext")
	}
}

func assertConfigIPCPrivate(t *testing.T, responses ...ipc.Response) {
	t.Helper()
	for _, response := range responses {
		body, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(body, []byte("canario")) || bytes.Contains(body, []byte(`"body"`)) || bytes.Contains(body, []byte(`"stamp"`)) {
			t.Fatal("configuration secret or stored envelope entered IPC")
		}
	}
}
