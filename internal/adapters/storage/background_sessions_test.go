package storage

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"testing"
	"time"
)

func TestBackgroundStoresOnlyCurrentAndEdgesForTwoFullConcentrators(t *testing.T) {
	j, a := ppStoreSetup(t)
	b := elsewherePeer(t, j, a)
	at := time.Now().UTC().Add(-2 * time.Minute)
	var ledgerBefore int
	if err := j.st.unit.QueryRow("SELECT count(*) FROM equipment_operations").Scan(&ledgerBefore); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		for _, target := range []equipment.PPTarget{a, b} {
			stamp := at.Add(time.Duration(cycle) * time.Minute)
			o := backgroundObservation(target, stamp, cycle)
			if code := j.st.SavePulseReading(context.Background(),
				companyX,
				target,
				pulse.Reading{At: stamp,
					Uptime:      time.Hour + time.Duration(cycle)*time.Minute,
					Observation: o}); code != equipment.OK {
				t.Fatal(cycle, code)
			}
		}
	}
	assertBackgroundHydration(t, j, a.Device.ID)
	for table, want := range map[string]int{"pppoe_current": 2,
		"pppoe_observations":   2,
		"pppoe_rows":           4096,
		"session_history":      4096,
		"equipment_operations": 1} {
		var count int
		if table == "equipment_operations" {
			want = ledgerBefore
		}
		if err := j.st.unit.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != want {
			t.Fatal(table, count, err)
		}
	}
}

func backgroundObservation(target equipment.PPTarget, stamp time.Time, cycle int) *equipment.PPObservation {
	o := &equipment.PPObservation{ID: uuid.NewString(), DeviceID: target.Device.ID, Scope: equipment.PPScope,
		Quality: equipment.PPQuality{Status: "complete", Terminated: true, MembershipChecked: true},
		Provenance: equipment.PPProvenance{Automatic: true, Installation: target.Installation, Company: companyX, DeviceID: target.Device.ID,
			Revision:           target.Device.Revision,
			CredentialRevision: target.Credential.Metadata.Revision,
			Connection:         equipment.PPConnection(target.Device),
			Projection:         "active-v1",
			Codec:              "pppoe-fields-v1",
			Capabilities:       "base-fields-v1",
			Started:            equipment.PPTime(stamp),
			Ended:              equipment.PPTime(stamp)}}
	sampleAt := int64((cycle+1)*60) * int64(time.Second)
	o.Provenance.Sample = equipment.PPInterval{Known: true, Min: sampleAt, Max: sampleAt + int64(time.Millisecond)}
	for i := 0; i < 2048; i++ {
		o.Rows = append(o.Rows,
			equipment.PPRow{ID: uuid.NewString(),
				RemoteID:  fmt.Sprint(i),
				SessionID: fmt.Sprint(i),
				Address:   "192.0.2.10", Caller: "02:00:00:00:00:01",
				Sample:   o.Provenance.Sample,
				Uptime:   equipment.PPInterval{Known: true, Min: int64(time.Hour), Max: int64(time.Hour + time.Second)},
				Download: &equipment.PPRate{Min: 999999.9999999999, Max: 1000000.0000000001},
				Upload:   &equipment.PPRate{Min: 499999.99999999994, Max: 500000.00000000006},
				Login: fmt.Sprintf("%s-%04d",
					target.Device.ID,
					i)})
	}
	return o
}

func assertBackgroundHydration(t *testing.T, j *journey, device string) {
	t.Helper()
	current, code := j.st.PPCurrent(context.Background(), companyX, device)
	if code != equipment.OK {
		t.Fatal(code)
	}
	page, code := j.st.PPPage(context.Background(), companyX, equipment.PPInput{DeviceID: device, ObservationID: current.Current.ID})
	if code != equipment.OK || len(page.Rows) != 16 {
		t.Fatal(code, page)
	}
	restored := page.Rows[0]
	if restored.Login == "" || restored.Address != "192.0.2.10" ||
		restored.Caller != "02:00:00:00:00:01" ||
		!restored.Sample.Known || restored.Download == nil || restored.Download.Max != 1000000.0000000001 {
		t.Fatal("row hydration lost data", restored)
	}
	tx, err := j.st.unit.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	linked, code := readRadioLinkSource(context.Background(), tx,
		equipment.RadioLink{Concentrator: device, Login: restored.Login},
		equipment.PPInput{ObservationID: current.Current.ID, RowID: restored.ID})
	if code != equipment.OK || linked.Address != "192.0.2.10" {
		t.Fatal("normalized session lost link source", code, linked)
	}

}
