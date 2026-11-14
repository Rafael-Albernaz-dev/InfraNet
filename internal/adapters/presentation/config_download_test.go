package presentation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
)

type configDownloadRemote struct{}

func (configDownloadRemote) ReadConfig(context.Context, equipment.Device, string, string, func() equipment.Code) (string, equipment.Code) {
	return "/ppp secret\nadd password=senha-canario\n", equipment.OK
}
func prepareConfigDownload(t *testing.T) (*operationalRig, string) {
	t.Helper()
	r := newOperationalRig(t)
	r.enrollDevice()
	r.acceptTrustWithoutPassword()
	code := r.st.UpdateEquipment(r.ctx, r.profile, func(state *equipment.State) equipment.Code {
		state.Devices[0].Result = &equipment.Observation{Family: equipment.RouterFamilyV6, Version: "6.49.10"}
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
	service := cfgbackup.New(r.auth, r.st, r.provider, configDownloadRemote{}, ConfigFiles{}, modalIDs{})
	r.server.AttachConfig(service)
	t.Cleanup(service.Close)
	view := r.app.SaveConfig(r.profile, r.activation, r.device.ID)
	if view.Code != "OK" {
		t.Fatal(view.Code)
	}
	deadline := time.Now().Add(3 * time.Second)
	for view.Data.Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
		view = r.app.ListConfig(r.profile, r.activation, r.device.ID)
	}
	if view.Code != "OK" || view.Data.Busy || len(view.Data.Versions) != 1 {
		t.Fatal("configuration fixture")
	}
	return r, view.Data.Versions[0].ID
}
func TestConfigNativeDownloadBindsConfirmationToSession(t *testing.T) {
	r, version := prepareConfigDownload(t)
	picked := false
	destination := filepath.Join(r.root, "confirmed.rsc")
	r.app.saveDialog = func(_ context.Context, options runtime.SaveDialogOptions) (string, error) {
		picked = true
		if options.Title != "Este arquivo contém senhas do equipamento" || options.DefaultFilename != "configuracao.rsc" {
			t.Error("native sensitive download dialog")
		}
		return destination, nil
	}
	if r.app.DownloadConfig(r.profile, r.activation, r.device.ID, version, false).Code != "INVALID_REQUEST" || picked {
		t.Fatal("unconfirmed native download")
	}
	if result := r.app.DownloadConfig(r.profile, r.activation, r.device.ID, version, true); result.Code != "OK" {
		t.Fatal(result.Code)
	}
	text, err := os.ReadFile(destination)
	if err != nil || string(text) != "/ppp secret\nadd password=senha-canario\n" {
		t.Fatal("confirmed export did not write full text")
	}
	destination = filepath.Join(r.root, "stale-confirmation.rsc")
	r.app.saveDialog = func(context.Context, runtime.SaveDialogOptions) (string, error) {
		if r.app.Logout().Code != "OK" || r.app.Login("technical", "isolated modal account password", false).Code != "OK" {
			t.Fatal("synthetic session switch")
		}
		return destination, nil
	}
	result := r.app.DownloadConfig(r.profile, r.activation, r.device.ID, version, true)
	if result.Code != "PROFILE_CHANGED" {
		t.Fatalf("old confirmation crossed logout/login: %s", result.Code)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatal("stale confirmation wrote plaintext")
	}
}
