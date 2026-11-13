package storage

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"testing"
	"time"
)

func TestCollectionGapPreservesOpenSessionAndRecordsItsOwnInterval(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	at := time.Now().UTC().Add(-15 * time.Minute)
	elsewherePublish(t, j, target, at, "synthetic")
	initial, code := j.st.PPSessionHistory(ctx, companyX, equipment.PPInput{DeviceID: target.Device.ID, Login: "synthetic"})
	if code != equipment.OK {
		t.Fatal(code)
	}
	if code := j.st.MarkPulseGap(ctx, companyX, target, at.Add(time.Minute)); code != equipment.OK {
		t.Fatal(code)
	}
	elsewherePublish(t, j, target, at.Add(11*time.Minute), "synthetic")
	view, code := j.st.PPSessionHistory(ctx, companyX, equipment.PPInput{DeviceID: target.Device.ID, Login: "synthetic"})
	if code != equipment.OK || len(view.History) != 2 {
		t.Fatal(code, view.History)
	}
	stale := equipment.PPInput{DeviceID: target.Device.ID, Login: "synthetic", HistoryGeneration: initial.HistoryGeneration}
	// ignored: Only the rejection code is relevant for the stale page.
	if _, code = j.st.PPSessionHistory(ctx, companyX, stale); code != equipment.CursorInvalid {
		t.Fatal("changed history accepted stale page", code)
	}
	var sessions, gaps int
	for _, r := range view.History {
		if r.Kind == "session" {
			sessions++
			if r.Disconnected != "" {
				t.Fatal("gap invented disconnect", r)
			}
		}
		if r.Kind == "gap" {
			gaps++
			if r.Connected == "" || r.Disconnected == "" || r.Reason != "Lacuna de coleta" {
				t.Fatal(r)
			}
		}
	}
	if sessions != 1 || gaps != 1 {
		t.Fatal(view.History)
	}
}

func TestSessionWithoutLogoutEvidenceUsesUnknownReason(t *testing.T) {
	j, target := ppStoreSetup(t)
	at := time.Now().UTC().Add(-time.Minute)
	elsewherePublish(t, j, target, at, "synthetic")
	elsewherePublish(t, j, target, at.Add(time.Minute))
	view, code := j.st.PPSessionHistory(context.Background(), companyX, equipment.PPInput{DeviceID: target.Device.ID, Login: "synthetic"})
	if code != equipment.OK ||
		len(view.History) != 1 ||
		view.History[0].Reason != "Não informado nesta leitura" ||
		view.History[0].Disconnected == "" {
		t.Fatal(code, view.History)
	}
}

func TestPeerOnlineImmediatelyRemovesOfflineWithoutEditingSavedReference(t *testing.T) {
	j, a := ppStoreSetup(t)
	b := elsewherePeer(t, j, a)
	at := time.Now().UTC().Add(-5 * time.Minute)
	elsewherePublish(t, j, a, at, "synthetic")
	offline := elsewherePublish(t, j, a, at.Add(time.Minute))
	if offline.Offline != 1 {
		t.Fatal("fixture not offline")
	}
	ref := equipment.PPInput{DeviceID: a.Device.ID, ObservationID: offline.ID, ReferenceID: uuid.NewString(), Operation: uuid.NewString()}
	// ignored: Only the status or membership is needed in this check.
	if _, code := j.st.SavePPReference(context.Background(), companyX, ref, "saved-offline"); code != equipment.OK {
		t.Fatal(code)
	}
	elsewherePublish(t, j, b, at.Add(2*time.Minute), "synthetic")
	current, code := j.st.PPCurrent(context.Background(), companyX, a.Device.ID)
	if code != equipment.OK || current.Current.Offline != 0 || current.Current.ID == offline.ID {
		t.Fatal(code, current.Current)
	}
	saved, code := j.st.PPPage(context.Background(), companyX, equipment.PPInput{DeviceID: a.Device.ID, ObservationID: offline.ID})
	if code != equipment.OK || len(saved.Rows) != 1 || saved.Rows[0].State != "OFFLINE" {
		t.Fatal("saved reference changed", code, saved)
	}
	// A saved reference retains the old immutable observation.
	var count int
	if err := j.st.unit.QueryRow("SELECT count(*) FROM session_history WHERE device_id=? AND login='synthetic'",
		a.Device.ID).Scan(&count); err != nil ||
		count != 1 {
		t.Fatal(count, err)
	}
}

func TestLastOnlineSurvivesStoreReopen(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	at := time.Now().UTC().Truncate(time.Second)
	values := map[string]pulse.DeviceStatus{target.Device.ID: {Address: target.Device.Address,
		State:     pulse.Online,
		Since:     at,
		LastReply: at,
		CheckedAt: at}}
	if err := j.st.SavePresence(ctx, companyX, values, at); err != nil {
		t.Fatal(err)
	}
	j.create(companyY, "Y", access.OK)
	j.selectCompany(companyY, access.OK)
	j.selectCompany(companyX, access.OK)
	loaded, err := j.st.LoadPresence(ctx, companyX)
	if err != nil || !loaded[target.Device.ID].LastReply.Equal(at) {
		t.Fatal(loaded, err)
	}
	state, code := j.st.ReadEquipment(ctx, companyX)
	if code != equipment.OK || state.Devices[0].Pulse == nil || state.Devices[0].Pulse.LastReply != equipment.PPTime(at) {
		t.Fatal(code, state.Devices)
	}
}

func TestBootCheckpointDeduplicatesLiveAndManualAfterDelayedRead(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	at := time.Now().UTC().Add(-time.Hour)
	save := func(at time.Time, up time.Duration) {
		t.Helper()
		code := j.st.SavePulseReading(ctx, companyX, target, pulse.Reading{At: at, Uptime: up})
		if code != equipment.OK {
			t.Fatal(code)
		}
	}
	save(at, 2*time.Hour)
	save(at.Add(30*time.Minute), time.Hour)
	h := &equipment.DeviceHealth{At: equipment.PPTime(at.Add(31 * time.Minute)),
		Started: equipment.PPTime(at.Add(31 * time.Minute)),
		Uptime: equipment.PPInterval{Known: true,
			Min: int64(61 * time.Minute),
			Max: int64(61*time.Minute + time.Second)}}
	r := &equipment.PPReadResult{Ended: h.At, Rows: []equipment.PPReadRow{{Kind: "resource", Health: h}}}
	if code := j.st.SaveDeviceMonitoring(ctx, companyX, target.Device, target.Credential.Metadata.Revision, r); code != equipment.OK {
		t.Fatal(code)
	}
	var n int
	if err := j.st.unit.QueryRow("SELECT count(*) FROM device_equipment_events WHERE kind='reboot'").Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
}

func TestEquipmentChangesRecordBeforeAndAfterOnlyOnce(t *testing.T) {
	j, target := ppStoreSetup(t)
	at := time.Now().UTC()
	for i, r := range []pulse.Reading{{Firmware: "7.1",
		Identity: "before",
		Model:    "A"},
		{Firmware: "7.2",
			Identity: "after",
			Model:    "B"},
		{Firmware: "7.2",
			Identity: "after",
			Model:    "B"}} {
		r.At = at.Add(time.Duration(i) * time.Minute)
		if code := j.st.SavePulseReading(context.Background(), companyX, target, r); code != equipment.OK {
			t.Fatal(code)
		}
	}
	var n int
	if err := j.st.unit.QueryRow("SELECT count(*) FROM device_pulse_events WHERE before_value<>'' AND after_value<>''").Scan(&n); err != nil ||
		n != 3 {
		t.Fatal(n, err)
	}
}

func TestPulseHistoryRetentionRemovesOnlyExpiredClosedRecords(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	now := time.Now().UTC()
	code := j.st.ppWrite(ctx, companyX, func(tx *sql.Tx) equipment.Code {
		for _, age := range []int{91, 89} {
			at := equipment.PPTime(now.Add(-time.Duration(age) * 24 * time.Hour))
			// ignored: Only the SQL error is consumed; the result carries no required domain value.
			// ignored: Only the SQL error is needed; no result value is consumed.
			if _, err := tx.Exec("INSERT INTO session_history(device_id,login,session_key,kind,connected_at,discon"+
				"nected_at,row_json) VALUES(?,'synthetic','','gap',?,?,'{}')", target.Device.ID, at, at); err != nil {
				t.Fatal(err)
			}
			if _,
				err := tx.Exec("INSERT INTO device_pulse_events(device_id,kind,started_at,ended_at) VALUES(?,'firmware',?,?)",
				target.Device.ID,
				at,
				at); err != nil {
				t.Fatal(err)
			}
		}
		for _, age := range []int{91, 89} {
			at := now.Add(-time.Duration(age) * 24 * time.Hour)
			if _,
				err := tx.Exec("INSERT INTO device_equipment_events VALUES(?,?,?,'reboot','','{}')",
				target.Device.ID,
				equipment.PPTime(at),
				at.Unix()); err != nil {
				t.Fatal(err)
			}
		}
		if err := pulseHistoryGC(ctx, tx, now); err != nil {
			t.Fatal(err)
		}
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
	for _, table := range []string{"session_history", "device_pulse_events", "device_equipment_events"} {
		var n int
		if err := j.st.unit.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 1 {
			t.Fatal(table, n, err)
		}
	}
}

func TestPingOutageRecordsOneIntervalFromLastReply(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	tracker := pulse.NewTracker()
	a := pulse.Target{ID: target.Device.ID, Address: target.Device.Address}
	tracker.Sync([]pulse.Target{a})
	at := time.Now().UTC().Truncate(time.Second)
	record := func(outcome pulse.Outcome, seconds int) {
		t.Helper()
		now := at.Add(time.Duration(seconds) * time.Second)
		tracker.Record(a, outcome, now)
		if err := j.st.SavePresence(ctx, companyX, tracker.Snapshot(now).Devices, now); err != nil {
			t.Fatal(err)
		}
	}
	record(pulse.Reply, 0)
	record(pulse.NoReply, 10)
	record(pulse.NoReply, 20)
	record(pulse.NoReply, 30)
	record(pulse.NoReply, 40)
	record(pulse.Reply, 50)
	var count int
	var start, end string
	if err := j.st.unit.QueryRow("SELECT count(*),started_at,ended_at FROM device_pulse_events WHERE kind='OFFLINE'").Scan(&count,
		&start,
		&end); err != nil ||
		count != 1 ||
		start != equipment.PPTime(at) ||
		end != equipment.PPTime(at.Add(50*time.Second)) {
		t.Fatal(count, start, end, err)
	}
}
