package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"testing"
	"time"
)

func TestLinkedInterfaceQueuesThirtyCPEsWithFourWorkersAndOneReadPerAP(t *testing.T) {
	f := newEnlaceFixture(t, 30)
	f.remote.entered = make(chan string, 30)
	f.remote.release = make(chan struct{})
	done := make(chan struct{})
	go func() { f.round(); close(done) }()
	for i := 0; i < 4; i++ {
		select {
		case <-f.remote.entered:
		case <-time.After(10 * time.Second):
			t.Fatal("four guarded CPE reads did not start")
		}
	}
	if f.remote.calls.Load() != 4 {
		t.Fatal("more than four CPEs admitted", f.remote.calls.Load())
	}
	row := f.observation.Rows[29]
	q := equipmentScope(f.j)
	q.Op = "radio.enlace"
	_, c := f.service.HandleRadio(context.Background(), q, equipment.RadioInput{
		DeviceID: f.target.Device.ID, ObservationID: f.observation.ID, RowID: row.ID})
	if c != equipment.OK {
		t.Fatal(c)
	}
	f.remote.release <- struct{}{}
	select {
	case name := <-f.remote.entered:
		if name != "CPE "+row.Login {
			t.Fatal("open session did not get next slot", name)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("priority CPE did not start")
	}
	close(f.remote.release)
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("queue stalled")
	}
	assertEnlaceQueueResults(t, f)
}
func assertEnlaceQueueResults(t *testing.T, f *enlaceFixture) {
	t.Helper()
	if f.remote.peak.Load() != 4 || f.remote.calls.Load() != 30 {
		t.Fatal("queue bounds", f.remote.peak.Load(), f.remote.calls.Load())
	}
	for _, ap := range f.aps {
		if f.remote.reads[ap] != 1 {
			t.Fatal("AP read per client", f.remote.reads)
		}
	}
	devices := f.devices(t)
	if len(devices) != 33 {
		t.Fatal("missing or duplicate CPEs", len(devices))
	}
	for i, row := range f.observation.Rows {
		links, c := f.j.st.RadioLinks(context.Background(), companyX, f.target.Device.ID, row.Login)
		if c != equipment.OK || len(links) != 1 || links[0].AP != f.aps[i%2] || links[0].Origin != "enlace" {
			t.Fatal("wrong AP association", c, links)
		}
		assertEnlaceInitialReading(t, f, links[0].CPE)
	}
}

func TestEnlaceAdoptsTheManuallyRegisteredCPEWithoutDuplicatingIt(t *testing.T) {
	f := newEnlaceFixture(t, 1)
	id := uuid.NewString()
	if c := f.j.st.UpdateEquipment(context.Background(), companyX, func(st *equipment.State) equipment.Code {
		d := f.target.Device
		d.ID, d.Role, d.Address, d.Fingerprint = id, equipment.RoleCPE, f.observation.Rows[0].Address, ""
		st.Devices = append(st.Devices, d)
		return equipment.OK
	}); c != equipment.OK {
		t.Fatal(c)
	}
	f.round()
	if len(f.devices(t)) != 4 || f.remote.calls.Load() != 1 {
		t.Fatal("manual CPE duplicated")
	}
	links, c := f.j.st.RadioLinks(context.Background(), companyX, f.target.Device.ID, f.observation.Rows[0].Login)
	if c != equipment.OK || len(links) != 1 || links[0].CPE != id {
		t.Fatal("did not adopt manual CPE", c, links)
	}
	for _, d := range f.devices(t) {
		if d.ID == id && (d.CPELogin == nil || d.CPELogin.Origin == "enlace") {
			t.Fatal("manual CPE mislabeled", d.CPELogin)
		}
	}
}

func TestEnlaceFillsOnlyTheRemainingInventoryCapacity(t *testing.T) {
	f := newEnlaceFixture(t, 20)
	if c := f.j.st.UpdateEquipment(context.Background(), companyX, func(st *equipment.State) equipment.Code {
		for len(st.Devices) < 4990 {
			n := len(st.Devices)
			st.Devices = append(st.Devices, equipment.Device{ID: uuid.NewString(), Name: fmt.Sprint(n), Address: "192.0.2.200", Port: 22,
				Role: equipment.RoleAP, Transport: equipment.TransportSSH, Revision: 1})
		}
		return equipment.OK
	}); c != equipment.OK {
		t.Fatal(c)
	}
	f.round()
	if len(f.devices(t)) != 5000 || f.remote.calls.Load() != 10 {
		t.Fatal("capacity not enforced per client", len(f.devices(t)), f.remote.calls.Load())
	}
	failed := 0
	for _, row := range f.observation.Rows {
		q := equipmentScope(f.j)
		q.Op = "radio.enlace"
		v, c := f.service.HandleRadio(context.Background(), q, equipment.RadioInput{
			DeviceID: f.target.Device.ID, ObservationID: f.observation.ID, RowID: row.ID})
		if c != equipment.OK || v.Enlace == nil {
			t.Fatal(c)
		}
		if v.Enlace.Reason == string(equipment.Limit) {
			failed++
		}
	}
	if failed != 10 {
		t.Fatal("capacity failure did not stay local to remaining clients", failed)
	}
	assertFullEnlaceInventory(t, f)

}

func TestReconnectedLoginMovesTheCPEAndRefusesAChangedKey(t *testing.T) {
	f := newEnlaceFixture(t, 1)
	f.round()
	var before equipment.Device
	for _, d := range f.devices(t) {
		if d.CPELogin != nil {
			before = d
		}
	}
	rows := append([]equipment.PPRow{}, f.observation.Rows...)
	rows[0].Address = "100.64.1.99"
	f.publish(t, rows)
	sessions, c := f.j.st.EnlaceSessions(context.Background(), companyX)
	if c != equipment.OK {
		t.Fatal(c)
	}
	if c = f.service.FollowEnlaceSessions(context.Background(), companyX, sessions); c != equipment.OK {
		t.Fatal(c)
	}
	for _, d := range f.devices(t) {
		if d.ID == before.ID && (d.Address != rows[0].Address || d.Fingerprint != before.Fingerprint || d.Revision <= before.Revision) {
			t.Fatal("reconnection lost IP/pin/revision", d.Address, d.Fingerprint, d.Revision)
		}
	}
	f.remote.fingerprint = "SHA256:unexpected-replacement"
	f.round()
	after := f.devices(t)
	assertEnlacePinRefused(t, after, before)
	calls := f.remote.calls.Load()
	f.round()
	if f.remote.calls.Load() != calls {
		t.Fatal("changed key was retried automatically")
	}
}

func TestOfflineLoginKeepsItsCPEAndLinkWithoutReadingIt(t *testing.T) {
	f := newEnlaceFixture(t, 1)
	f.round()
	calls := f.remote.calls.Load()
	lastOnline := f.observation.Provenance.Ended
	rows := append([]equipment.PPRow{}, f.observation.Rows...)
	rows[0].State = "OFFLINE"
	f.publish(t, nil)
	f.round()
	links, c := f.j.st.RadioLinks(context.Background(), companyX, f.target.Device.ID, rows[0].Login)
	if c != equipment.OK || len(links) != 1 || len(f.devices(t)) != 4 || f.remote.calls.Load() != calls {
		t.Fatal("offline read or lost durable association", c, links)
	}
	for _, d := range f.devices(t) {
		if d.CPELogin != nil {
			assertOfflineLoginPresence(t, f, d, lastOnline)
		}
	}
	q := equipmentScope(f.j)
	q.Op = "radio.enlace"
	v, c := f.service.HandleRadio(context.Background(), q, equipment.RadioInput{
		DeviceID: f.target.Device.ID, ObservationID: f.observation.ID, RowID: rows[0].ID})
	if c != equipment.OK || v.Enlace == nil || v.Enlace.State != "offline" {
		t.Fatal("offline presentation", c, v.Enlace)
	}
}

func assertOfflineLoginPresence(t *testing.T, f *enlaceFixture, d equipment.Device, lastOnline string) {
	t.Helper()
	presence := d.CPELogin.Session
	if presence == nil || presence.State != "OFFLINE" || presence.LastOnline != lastOnline ||
		presence.ReadAt != f.observation.Provenance.Ended || presence.Since == "" {
		t.Fatal("persisted session presence or last ONLINE collection lost", presence)
	}
	q := equipmentScope(f.j)
	q.Op, q.ID = "inventory.get", d.ID
	view, code := f.service.Service.Handle(context.Background(), q)
	if code != equipment.OK || view.Device == nil || view.Device.Pulse == nil {
		t.Fatal("CPE detail unavailable", code, view.Device)
	}
	want := equipment.PulseStatus{State: "OFFLINE", Since: presence.Since,
		LastReply: lastOnline, CheckedAt: presence.ReadAt}
	if *view.Device.Pulse != want {
		t.Fatal("CPE summary lost session presence", view.Device.Pulse, want)
	}
}

func TestRefusedCPEPasswordDoesNotRetryUntilTheCredentialChanges(t *testing.T) {
	f := newEnlaceFixture(t, 1)
	f.remote.outcome = equipment.RadioCodeRemoteAuthFailed
	f.round()
	f.round()
	if f.remote.calls.Load() != 1 {
		t.Fatal("refused password retried", f.remote.calls.Load())
	}
	if c := f.j.st.UpdateEquipment(context.Background(), companyX, func(st *equipment.State) equipment.Code {
		cr := &st.Credentials[0]
		cr.Metadata.Revision++
		v := vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil })
		envelope, wrapped, code := v.Seal(st.Installation, companyX, cr.Metadata.ID, cr.Metadata.Revision, sshfixture.Password, st.WrappedKey)
		if code != equipment.OK {
			return code
		}
		cr.Envelope, st.WrappedKey = envelope, wrapped
		return equipment.OK
	}); c != equipment.OK {
		t.Fatal(c)
	}
	f.remote.outcome = equipment.OK
	f.round()
	if f.remote.calls.Load() != 2 {
		t.Fatal("credential change did not release retry", f.remote.calls.Load())
	}
}

func assertEnlacePinRefused(t *testing.T, after []equipment.Device, before equipment.Device) {
	t.Helper()
	for _, d := range after {
		if d.ID == before.ID {
			if d.Fingerprint != before.Fingerprint || d.LastAttempt == nil || d.LastAttempt.Code != equipment.TrustChanged {
				t.Fatal("changed key accepted", d.LastAttempt)
			}
		}
	}
}

func assertFullEnlaceInventory(t *testing.T, f *enlaceFixture) {
	t.Helper()
	q := equipmentScope(f.j)
	q.Op = "inventory.list"
	q.All = true
	count := 0
	for {
		v, c := f.service.Handle(context.Background(), q)
		if c != equipment.OK {
			t.Fatal(c)
		}
		count += len(v.Devices)
		if v.Next == 0 {
			break
		}
		q.Cursor = v.Next
	}
	if count != 5000 {
		t.Fatal("inventory cannot list its full capacity", count)
	}
}

func assertEnlaceInitialReading(t *testing.T, f *enlaceFixture, id string) {
	t.Helper()
	// ignored: this assertion concerns the saved initial reading, not its driver profile.
	reading, _, c := f.j.st.RadioCurrent(context.Background(), companyX, id)
	if c != equipment.OK || reading == nil || !reading.Compatible || reading.Model != "synthetic CPE" || reading.Version != "6.synthetic" {
		t.Fatal("initial CPE radio reading not published", c, reading)
	}
	if len(reading.Metrics) != 1 || reading.Metrics[0].Name != "signal" || reading.Metrics[0].Value != -61 {
		t.Fatal("initial signal lost", reading.Metrics)
	}

}
