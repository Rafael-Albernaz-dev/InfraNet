package storage

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/modules/equipment"
	"testing"
	"time"
)

func TestSessionTableQueriesTheEntireObservationBeforePaging(t *testing.T) {
	j, target := ppStoreSetup(t)
	o := largeSessionTableFixture(t, j, target)
	seedSessionTableColumns(t, j, o)
	in := equipment.PPInput{DeviceID: target.Device.ID,
		ObservationID: o.ID,
		Query: &equipment.PPQuery{PageSize: 200,
			Search:  "rb",
			Plan:    "Árvore",
			VLAN:    "7",
			Service: "Serviço",
			Sort:    "caller_id",
			Desc:    true}}
	v, c := j.st.PPPage(context.Background(), companyX, in)
	if c != equipment.OK || v.Total != 1024 || len(v.Rows) != 200 || !v.More || v.NextOffset != 200 {
		t.Fatal("query was restricted to page", c, v.Total, len(v.Rows))
	}
	if len(v.Options["plan"]) != 2 || len(v.Options["vlan"]) != 2 {
		t.Fatal("dropdowns were restricted to filtered page", v.Options)
	}
	in.Offset = 1000
	v, c = j.st.PPPage(context.Background(), companyX, in)
	if c != equipment.OK || len(v.Rows) != 24 || v.More {
		t.Fatal("last page", c, len(v.Rows))
	}
	for _, row := range v.Rows {
		if row.Context == nil || row.Context.VLAN == nil || *row.Context.VLAN != 7 {
			t.Fatal("filter escaped", row)
		}
	}
}

func TestSessionTableOrdersNumbersAndKeepsMissingValuesLast(t *testing.T) {
	j, target := ppStoreSetup(t)
	o := ppSynthetic(t, j, target, 4, "synthetic")
	for i, vlan := range []any{nil, 100, 2, nil} {
		_,
			err := j.st.unit.Exec("UPDATE pppoe_rows SET row_json=json_set(row_json,'$.context.vlan',?,'$.context.profile',?) "+
			"WHERE observation_id=? AND row_id=?",
			vlan,
			fmt.Sprint(i),
			o.ID,
			o.Rows[i].ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, desc := range []bool{false, true} {
		in := equipment.PPInput{DeviceID: target.Device.ID,
			ObservationID: o.ID,
			Query: &equipment.PPQuery{PageSize: 25,
				Sort: "vlan",
				Desc: desc}}
		v, c := j.st.PPPage(context.Background(), companyX, in)
		want := 2
		if desc {
			want = 100
		}
		if c != equipment.OK || len(v.Rows) != 4 || v.Rows[0].Context == nil || *v.Rows[0].Context.VLAN != want || v.Rows[3].Context.VLAN != nil {
			t.Fatal("numeric or missing order", c, v.Rows)
		}
	}
}

func largeSessionTableFixture(t *testing.T, j *journey, target equipment.PPTarget) *equipment.PPObservation {
	t.Helper()
	ctx := context.Background()
	job := uuid.NewString()
	in := equipment.PPInput{DeviceID: target.Device.ID, Revision: target.Device.Revision, Operation: uuid.NewString()}
	if _, _, c := j.st.ReservePPCollection(ctx, companyX, in, "synthetic:"+job, job); c != equipment.OK {
		t.Fatal(c)
	}
	o := &equipment.PPObservation{ID: uuid.NewString(),
		DeviceID:   target.Device.ID,
		Scope:      equipment.PPScope,
		Provenance: equipment.PPProvenance{Ended: time.Now().UTC().Format(time.RFC3339Nano)},
		Quality: equipment.PPQuality{Status: "complete",
			Terminated:        true,
			MembershipChecked: true}}
	for i := 0; i < 2048; i++ {
		o.Rows = append(o.Rows, equipment.PPRow{ID: fmt.Sprint(i), Login: "s", RemoteID: fmt.Sprint(i)})
	}
	a := equipment.PPAttempt{JobID: job, Code: equipment.OK, State: "SUCCEEDED", ObservationID: o.ID}
	if c := j.st.FinishPPCollection(ctx, companyX, target, job, a, o); c != equipment.OK {
		t.Fatal(c)
	}
	return o
}

func seedSessionTableColumns(t *testing.T, j *journey, o *equipment.PPObservation) {
	t.Helper()
	for i, row := range o.Rows {
		plan := "Plano 20"
		vlan := 20
		service := "pppoe"
		caller := "normal"
		if i%2 == 0 {
			plan = "Árvore"
			vlan = 7
			service = "Serviço"
			caller = "RB oculto"
		}
		const update = `UPDATE pppoe_rows SET row_json=json_set(row_json,
 '$.context.secret_state','active','$.context.profile',?,'$.context.vlan',?,
 '$.context.service',?,'$.caller_id',?),caller_id=? WHERE observation_id=? AND row_id=?`
		_, err := j.st.unit.Exec(update, plan, vlan, service, caller, caller, o.ID, row.ID)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCurrentTableRefusesAContinuationAfterAnotherObservationPublishes(t *testing.T) {
	j, target := ppStoreSetup(t)
	before := ppSynthetic(t, j, target, 50, "synthetic")
	svc := newEquipment(t, j)
	q := equipmentScope(j)
	q.Op = "pppoe.current"
	input := equipment.PPInput{DeviceID: target.Device.ID, Query: &equipment.PPQuery{PageSize: 25}}
	first, c := svc.HandlePP(context.Background(), q, input)
	if c != equipment.OK || len(first.Rows) != 25 || !first.More {
		t.Fatal("first page", c)
	}
	input.ObservationID = before.ID
	input.Offset = 25
	same, c := svc.HandlePP(context.Background(), q, input)
	if c != equipment.OK || same.Current.ID != before.ID || len(same.Rows) != 25 {
		t.Fatal("legitimate continuation", c)
	}
	after := ppSynthetic(t, j, target, 1, "synthetic")
	if after.ID == before.ID {
		t.Fatal("fixture did not publish a new observation")
	}
	if _, c = svc.HandlePP(context.Background(), q, input); c != equipment.CursorInvalid {
		t.Fatal("mixed observations accepted", c)
	}
}

func TestSessionTableSearchIncludesTheOnlineTextOfLegacyRows(t *testing.T) {
	j, target := ppStoreSetup(t)
	o := ppSynthetic(t, j, target, 1, "synthetic")
	// Recreate an observation written before the explicit state field existed.
	const legacyRow = "UPDATE pppoe_rows SET row_json=json_remove(row_json,'$.state') WHERE observation_id=?"
	// ignored: the update result is not needed; the fixture is verified by the public query below.
	if _, err := j.st.unit.Exec(legacyRow, o.ID); err != nil {
		t.Fatal(err)
	}
	input := equipment.PPInput{DeviceID: target.Device.ID, ObservationID: o.ID,
		Query: &equipment.PPQuery{PageSize: 25, Search: "ONLINE"}}
	view, c := j.st.PPPage(context.Background(), companyX, input)
	if c != equipment.OK || view.Total != 1 || len(view.Rows) != 1 || view.Rows[0].State != "ONLINE" {
		t.Fatal("legacy row displays ONLINE but cannot be found by its displayed text", c, view.Total)
	}
}
