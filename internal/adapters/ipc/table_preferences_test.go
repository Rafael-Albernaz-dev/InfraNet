package ipc

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/foundation"
	"reflect"
	"strings"
	"testing"
)

func TestTableArrangementIsAuthorizedAndBoundToActiveCompany(t *testing.T) {
	j := newPPJourney(t)
	client := j.client
	p := equipment.TablePreference{Columns: []string{"state"}, Filters: map[string]string{"state": "OFFLINE"}, PageSize: 100}
	q := Request{Op: "inventory.table-preference-set", ExpectedProfile: client.profile, ExpectedActivation: client.activation,
		Equipment: &EquipmentInput{Table: "equipment", TablePreference: &p}}
	r := client.Call(context.Background(), q)
	if r.Code != foundation.OK || r.Data == nil || !reflect.DeepEqual(r.Data.Equipment.TablePreference, &p) {
		t.Fatal("set", r.Code)
	}
	bad := p
	bad.PageSize = 30
	q.Equipment.TablePreference = &bad
	if r = client.Call(context.Background(), q); r.Code != foundation.Invalid {
		t.Fatal("invalid preference accepted", r.Code)
	}
	q.Op = "inventory.table-preference-get"
	q.Equipment = &EquipmentInput{Table: "equipment"}
	r = client.Call(context.Background(), q)
	if r.Code != foundation.OK || !reflect.DeepEqual(r.Data.Equipment.TablePreference, &p) {
		t.Fatal("invalid write changed stored arrangement", r.Code)
	}
	old := q
	r = client.Call(context.Background(), Request{Op: "company.create", Name: "Y", Operation: uuid.NewString()})
	if r.Code != foundation.OK {
		t.Fatal(r.Code)
	}
	r = client.Call(context.Background(), Request{Op: "access.bootstrap"})
	if r.Code != foundation.OK || len(r.Data.Companies) < 2 {
		t.Fatal("company catalog", r.Code)
	}
	r = client.Call(context.Background(), Request{Op: "company.select", Name: r.Data.Companies[1].ID})
	if r.Code != foundation.OK {
		t.Fatal(r.Code)
	}
	if r = client.Call(context.Background(), old); r.Code != foundation.ProfileChanged {
		t.Fatal("stale scope accepted", r.Code)
	}
	q.ExpectedProfile = client.profile
	q.ExpectedActivation = client.activation
	r = client.Call(context.Background(), q)
	if r.Code != foundation.OK || r.Data.Equipment.TablePreference != nil {
		t.Fatal("X preference leaked into Y", r.Code)
	}
}

func TestCurrentTableCursorPinsItsObservation(t *testing.T) {
	profile := uuid.NewString()
	q := Request{Op: "pppoe.current", Profile: profile, Activation: "active", PPPoE: &equipment.PPInput{
		DeviceID: uuid.NewString(), ObservationID: uuid.NewString(), Query: &equipment.PPQuery{PageSize: 25}}}
	server := &Server{}
	cursor := server.ppCursor(q, 25)
	old := q.PPPoE.ObservationID
	q.PPPoE.ObservationID = ""
	q.PPPoE.Cursor = cursor
	offset, ok := server.ppOffset(q)
	if !ok || offset != 25 || q.PPPoE.ObservationID != old {
		t.Fatal("cursor lost its source observation")
	}
	q.PPPoE.Query.PageSize = 50
	if _, ok = server.ppOffset(q); ok {
		t.Fatal("cursor crossed page size")
	}
}

func TestPreferenceWireRejectsAmbiguousNestedKeysAndAcceptsEmptyOptionalValues(t *testing.T) {
	id := uuid.NewString()
	q := Request{Version: 1, RequestID: "table", ContractRevision: ProductRevision, Op: "inventory.table-preference-set",
		Capability: strings.Repeat("a", 64), Session: "s", Profile: id, Activation: "active", ExpectedProfile: id, ExpectedActivation: "active",
		Equipment: &EquipmentInput{Table: "equipment", TablePreference: &equipment.TablePreference{PageSize: 50}}}
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	if _, c := validate(raw); c != foundation.OK {
		t.Fatal("empty optional values", c)
	}
	for _, body := range []string{
		`{"columns":[],"filters":{"state":"ONLINE","state":"OFFLINE"},"page_size":50}`,
		`{"columns":[],"page_size":25,"page_size":50}`,
		`{"sort":{"column":"login","column":"state"},"page_size":50}`,
	} {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			t.Fatal("fixture encoding")
		}
		obj["equipment"] = json.RawMessage(`{"table":"equipment","table_preference":` + body + `}`)
		bad, err := json.Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}
		if _, c := validate(bad); c != foundation.Invalid {
			t.Fatal("ambiguous preference accepted", body, c)
		}
	}
}
