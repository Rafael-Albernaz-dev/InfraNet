package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/radio"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type enlaceRemote struct {
	fingerprint string
	equipment.Remote
	mu                      sync.Mutex
	aps                     map[string][]equipment.RadioIndexEntry
	reads                   map[string]int
	entered                 chan string
	release                 chan struct{}
	calls, concurrent, peak atomic.Int32
	outcome                 equipment.Code
}

func (r *enlaceRemote) ReadRadioIndex(_ context.Context, d equipment.Device, _, _ string,
	guard func() equipment.Code,
) ([]equipment.RadioIndexEntry, equipment.Code) {
	if c := guard(); c != equipment.OK {
		return nil, c
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads[d.ID]++
	return r.aps[d.ID], equipment.OK
}
func (r *enlaceRemote) FirstContact(ctx context.Context, d equipment.Device, _, _ string,
	guard func() equipment.Code,
) (*equipment.ContactResult, equipment.Code) {
	if c := guard(); c != equipment.OK {
		return nil, c
	}
	n := r.concurrent.Add(1)
	defer r.concurrent.Add(-1)
	for p := r.peak.Load(); n > p && !r.peak.CompareAndSwap(p, n); p = r.peak.Load() {
	}
	r.calls.Add(1)
	if r.entered != nil {
		r.entered <- d.Name
	}
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return nil, equipment.Cancelled
		}
	}
	if r.outcome != "" && r.outcome != equipment.OK {
		return nil, r.outcome
	}
	fingerprint := r.fingerprint
	if fingerprint == "" {
		fingerprint = "SHA256:enlace-synthetic"
	}
	return &equipment.ContactResult{Fingerprint: fingerprint, Observation: &equipment.Observation{
		Identity: "synthetic CPE", Board: "synthetic", Version: "6.synthetic", At: equipment.PPTime(time.Now())}}, equipment.OK
}

type enlaceFixture struct {
	j           *journey
	target      equipment.PPTarget
	service     *radio.Service
	remote      *enlaceRemote
	aps         []string
	observation *equipment.PPObservation
}

func newEnlaceFixture(t *testing.T, n int) *enlaceFixture {
	t.Helper()
	j, target := ppStoreSetup(t)
	target.Device.Role = equipment.RoleConcentrator
	f := &enlaceFixture{j: j, target: target, aps: []string{uuid.NewString(), uuid.NewString()},
		remote: &enlaceRemote{aps: map[string][]equipment.RadioIndexEntry{}, reads: map[string]int{}}}
	if c := j.st.UpdateEquipment(context.Background(), companyX, func(st *equipment.State) equipment.Code {
		st.Devices[0].Role = equipment.RoleConcentrator
		st.CredentialDefaults[string(equipment.RoleCPE)] = target.Device.CredentialID
		for _, id := range f.aps {
			d := target.Device
			d.ID, d.Role, d.Fingerprint = id, equipment.RoleAP, "SHA256:synthetic-ap"
			st.Devices = append(st.Devices, d)
		}
		return equipment.OK
	}); c != equipment.OK {
		t.Fatal(c)
	}
	f.saveCatalog(t)
	if c := j.st.SetInterfaceAPs(context.Background(), companyX, equipment.InterfaceAPs{
		Concentrator: target.Device.ID, Interface: "vlan1004", APs: f.aps}); c != equipment.OK {
		t.Fatal(c)
	}
	rows := []equipment.PPRow{}
	for i := 0; i < n; i++ {
		mac := fmt.Sprintf("02:00:00:00:%02x:%02x", i/256, i%256)
		row := equipment.PPRow{ID: uuid.NewString(), RemoteID: fmt.Sprint(i), SessionID: fmt.Sprint(i),
			Login: fmt.Sprintf("client-%04d", i), Address: fmt.Sprintf("100.64.%d.%d", i/250, i%250+1), Caller: mac, State: "ONLINE",
			Context: &equipment.PPContext{Association: "associated", Service: "server-north", Interface: "<pppoe-synthetic>"}}
		rows = append(rows, row)
		id := f.aps[i%2]
		f.remote.aps[id] = append(f.remote.aps[id], equipment.RadioIndexEntry{MAC: mac, Interface: "wlan1", BSSID: "02:00:00:ff:00:01"})
	}
	f.publish(t, rows)
	f.service = radio.NewWithInventory(j.svc, j.st, vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil }),
		f.remote, equipmentIDs{}, time.Now)
	t.Cleanup(f.service.Close)
	return f
}
func (f *enlaceFixture) saveCatalog(t *testing.T) {
	t.Helper()
	r := &equipment.PPReadResult{Scope: equipment.PPPCatalogScope, Ended: equipment.PPTime(time.Now()), Rows: []equipment.PPReadRow{{
		Entry: &equipment.PPPEntry{ID: "server", Kind: "server", DeviceID: f.target.Device.ID, Name: "server-north",
			Service: "server-north", Interface: "vlan1004"}}}}
	if c := f.j.st.SavePPPCatalog(context.Background(), companyX, f.target.Device,
		f.target.Credential.Metadata.Revision, r); c != equipment.OK {
		t.Fatal(c)
	}
}
func (f *enlaceFixture) publish(t *testing.T, rows []equipment.PPRow) {
	t.Helper()
	job := uuid.NewString()
	in := equipment.PPInput{DeviceID: f.target.Device.ID, Revision: f.target.Device.Revision, Operation: uuid.NewString()}
	if _, _, c := f.j.st.ReservePPCollection(context.Background(), companyX, in, "enlace:"+job, job); c != equipment.OK {
		t.Fatal(c)
	}
	o := &equipment.PPObservation{ID: uuid.NewString(), DeviceID: f.target.Device.ID, Scope: equipment.PPScope, Rows: rows,
		Quality: equipment.PPQuality{Status: "complete", Terminated: true, MembershipChecked: true},
		Provenance: equipment.PPProvenance{Company: companyX, Installation: f.target.Installation, Projection: "synthetic", Codec: "synthetic",
			Capabilities: "synthetic", Ended: equipment.PPTime(time.Now())}}
	a := equipment.PPAttempt{JobID: job, Code: equipment.OK, State: "SUCCEEDED", ObservationID: o.ID}
	if c := f.j.st.FinishPPCollection(context.Background(), companyX, f.target, job, a, o); c != equipment.OK {
		t.Fatal(c)
	}
	f.observation = o
}
func (f *enlaceFixture) round() {
	f.service.EnlaceRound(context.Background(), companyX, func() bool { return true })
}
func (f *enlaceFixture) devices(t *testing.T) []equipment.Device {
	t.Helper()
	st, c := f.j.st.ReadEquipment(context.Background(), companyX)
	if c != equipment.OK {
		t.Fatal(c)
	}
	return st.Devices
}

func (r *enlaceRemote) ReadRadio(_ context.Context, d equipment.Device, _, _ string,
	_ equipment.RadioFamily, guard func() equipment.Code,
) (*equipment.RadioReading, equipment.Code) {
	if c := guard(); c != equipment.OK {
		return nil, c
	}
	return &equipment.RadioReading{DeviceID: d.ID, Family: equipment.RadioRouterOSWirelessV6, Version: "6.synthetic", Model: "synthetic CPE",
		Metrics: []equipment.RadioMetric{{Name: "signal", Value: -61, Unit: "dBm", Quality: "complete"}}}, equipment.OK
}
