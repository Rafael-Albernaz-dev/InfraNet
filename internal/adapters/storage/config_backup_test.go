package storage

import (
	"bytes"
	"context"
	"fmt"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type configFixture struct {
	j      *journey
	s      *cfgbackup.Service
	device equipment.Device
	text   atomic.Value
}

func newConfigFixture(t *testing.T, config sshfixture.Config) *configFixture {
	t.Helper()
	f := &configFixture{}
	f.text.Store("# nov/14/2026 03:10:22 by RouterOS 6.49.10\n/ppp secret\nadd password=senha-canario\n")
	if config.Command == nil {
		config.Command = func(command string) (string, bool) {
			if command == "/export" || command == "/export show-sensitive" {
				return f.text.Load().(string), true
			}
			return "", false
		}
	}
	peer, err := sshfixture.Start(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(peer.Close)
	f.j = newJourney(t)
	f.j.create(companyX, "X", access.OK)
	f.j.selectCompany(companyX, access.OK)
	inv := newEquipment(t, f.j)
	_, input := setupEquipment(t, f.j, inv, peer.Port())
	code := f.j.st.UpdateEquipment(context.Background(), companyX, func(st *equipment.State) equipment.Code {
		for i := range st.Devices {
			if st.Devices[i].ID == input.ID {
				d := &st.Devices[i]
				d.Fingerprint = peer.Fingerprint()
				d.Result = &equipment.Observation{Family: equipment.RouterFamilyV6, Version: "6.49.10"}
				f.device = *d
			}
		}
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
	v := vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil })
	f.s = cfgbackup.New(f.j.svc, f.j.st, v, routerssh.Adapter{}, nil, equipmentIDs{})
	t.Cleanup(f.s.Close)
	return f
}
func (f *configFixture) save(t *testing.T) cfgbackup.View {
	t.Helper()
	q := equipmentScope(f.j)
	q.Op = "config.save"
	v, c := f.s.Handle(context.Background(), q, cfgbackup.Input{DeviceID: f.device.ID})
	if c != equipment.OK {
		t.Fatal(c)
	}
	q.Op = "config.list"
	deadline := time.Now().Add(3 * time.Second)
	for v.Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
		v, c = f.s.Handle(context.Background(), q, cfgbackup.Input{DeviceID: f.device.ID})
		if c != equipment.OK {
			t.Fatal(c)
		}
	}
	if v.Busy {
		t.Fatal("configuration job did not finish")
	}
	return v
}
func TestConfigUnchangedDailyRound(t *testing.T) {
	f := newConfigFixture(t, sshfixture.Config{})
	at := time.Date(2026, 11, 14, 2, 0, 0, 0, time.Local)
	f.s.Now = func() time.Time { return at }
	manual := f.save(t)
	if manual.Status.Code != equipment.OK {
		t.Fatal(manual.Status.Code)
	}
	at = at.Add(70 * time.Minute)
	active := func() (string, bool) { return companyX, true }
	f.s.Round(context.Background(), active)
	automatic, c := f.j.st.ConfigStatus(context.Background(), companyX, f.device.ID)
	if c != equipment.OK || automatic.AutoAt == "" {
		t.Fatal("manual copy suppressed automatic round")
	}
	at = at.Add(24 * time.Hour)
	f.text.Store("# nov/15/2026 03:10:09 by RouterOS 6.49.10\n/ppp secret\nadd password=senha-canario\n")
	f.s.Round(context.Background(), active)
	versions, c := f.j.st.ConfigVersions(context.Background(), companyX, f.device.ID)
	status, sc := f.j.st.ConfigStatus(context.Background(), companyX, f.device.ID)
	if c != equipment.OK || sc != equipment.OK || len(versions) != 1 || !status.Unchanged {
		t.Fatal("timestamp created a version")
	}
	// Same local day must not query again; the status is durable across daemon restart.
	f.text.Store("changed\n")
	f.s.Round(context.Background(), active)
	versions, _ = f.j.st.ConfigVersions(context.Background(), companyX, f.device.ID)
	if len(versions) != 1 {
		t.Fatal("second automatic attempt on the same day")
	}
}
func TestConfigRetainsThirtyDifferentVersions(t *testing.T) {
	f := newConfigFixture(t, sshfixture.Config{})
	for i := 0; i < 31; i++ {
		f.text.Store(fmt.Sprintf("/system identity\nset name=version-%d\n", i))
		v := f.save(t)
		if v.Status.Code != equipment.OK || len(v.Versions) != min(i+1, 30) {
			t.Fatal("retention failed", v.Status.Code)
		}
	}
	versions, c := f.j.st.ConfigVersions(context.Background(), companyX, f.device.ID)
	if c != equipment.OK {
		t.Fatal(c)
	}
	st, _ := f.j.st.ReadEquipment(context.Background(), companyX)
	stored, c := f.j.st.ConfigVersion(context.Background(), companyX, f.device.ID, versions[29].ID)
	if c != equipment.OK {
		t.Fatal(c)
	}
	oldest, c := f.s.Vault.OpenConfig(st.Installation, companyX, versions[29].ID, stored.Body, st.WrappedKey)
	if c != equipment.OK || !strings.Contains(oldest, "version-1\n") {
		t.Fatal("wrong version pruned")
	}
}
func TestConfigRouterSevenExportsSensitive(t *testing.T) {
	var command atomic.Value
	f := newConfigFixture(t, sshfixture.Config{Command: func(cmd string) (string, bool) {
		command.Store(cmd)
		return "# 2026-11-14 03:10:22 by RouterOS 7.16.1\n/ppp secret\nadd password=senha-canario\n", true
	}})
	code := f.j.st.UpdateEquipment(context.Background(), companyX, func(st *equipment.State) equipment.Code {
		st.Devices[0].Result.Version = "7.16.1"
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
	v := f.save(t)
	if v.Status.Code != equipment.OK || command.Load() != "/export show-sensitive" || len(v.Versions) != 1 {
		t.Fatal("RouterOS 7 used an incomplete export")
	}
}
func TestConfigInterruptedExportNeverBecomesVersion(t *testing.T) {
	f := newConfigFixture(t, sshfixture.Config{Stream: func(cmd string, w io.ReadWriteCloser) bool {
		if cmd != "/export" {
			return false
		}
		// The output has a complete line, but SSH closes without exit status.
		if _, err := io.WriteString(w, "/ppp secret\nadd password=senha-canario\n"); err != nil {
			return true
		}
		return true
	}})
	v := f.save(t)
	if len(v.Versions) != 0 || v.Status.Code == equipment.OK {
		t.Fatal("partial export persisted")
	}
}
func TestConfigSQLiteContainsOnlyAuthenticatedCiphertext(t *testing.T) {
	f := newConfigFixture(t, sshfixture.Config{})
	v := f.save(t)
	if v.Status.Code != equipment.OK || len(v.Versions) != 1 {
		t.Fatal(v.Status.Code)
	}
	versions, c := f.j.st.ConfigVersions(context.Background(), companyX, f.device.ID)
	if c != equipment.OK {
		t.Fatal(c)
	}
	stored, c := f.j.st.ConfigVersion(context.Background(), companyX, f.device.ID, versions[0].ID)
	if c != equipment.OK {
		t.Fatal(c)
	}
	st, _ := f.j.st.ReadEquipment(context.Background(), companyX)
	text, c := f.s.Vault.OpenConfig(st.Installation, companyX, versions[0].ID, stored.Body, st.WrappedKey)
	if c != equipment.OK || !strings.Contains(text, "senha-canario") {
		t.Fatal("encrypted round trip failed")
	}
	if _, code := f.s.Vault.Open(st.Installation, companyX, versions[0].ID, 1, stored.Body, st.WrappedKey); code == equipment.OK {
		t.Fatal("configuration accepted under credential purpose")
	}
	if _, code := f.s.Vault.OpenConfig(st.Installation, companyX, "different-version", stored.Body, st.WrappedKey); code == equipment.OK {
		t.Fatal("version AAD not bound")
	}
	err := filepath.WalkDir(f.j.st.dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(body, []byte("senha-canario")) {
			t.Error("plaintext found in temporary SQLite storage")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestConfigUnavailableVaultNeverPersistsPlaintext(t *testing.T) {
	f := newConfigFixture(t, sshfixture.Config{})
	f.s.Vault = vault.New(func() ([]byte, error) { return nil, fmt.Errorf("unavailable") })
	v := f.save(t)
	if v.Status.Code != equipment.VaultUnavailable || len(v.Versions) != 0 {
		t.Fatal("missing key did not fail closed")
	}
}
