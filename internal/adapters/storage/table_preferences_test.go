package storage

import (
	"context"
	"fmt"
	"infradesktop/internal/adapters/security"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/inventory"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTableArrangementSurvivesCompanySwitchAndReopen(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.create(companyY, "Y", access.OK)
	j.selectCompany(companyX, access.OK)
	ctx := context.Background()
	x := inventory.TablePreference{Columns: []string{"state",
		"cpu"},
		Filters: map[string]string{"state": "OFFLINE"},
		Search:  "RB",
		Sort: &inventory.TableSort{Column: "cpu",
			Desc: true},
		Manual: []string{"b",
			"a"},
		PageSize: 100}
	if c := j.st.SetTablePreference(ctx, companyX, "equipment", x); c != inventory.OK {
		t.Fatal(c)
	}
	j.selectCompany(companyY, access.OK)
	if got, c := j.st.GetTablePreference(ctx, companyY, "equipment"); c != inventory.OK || got != nil {
		t.Fatal("X leaked into Y", got, c)
	}
	if c := j.st.SetTablePreference(ctx, companyX, "equipment", x); c != inventory.Changed {
		t.Fatal("inactive company write", c)
	}
	y := inventory.TablePreference{Search: "Y", PageSize: 25}
	if c := j.st.SetTablePreference(ctx, companyY, "equipment", y); c != inventory.OK {
		t.Fatal(c)
	}
	dir := j.st.dir
	j.st.Close()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	j.st = store
	j.svc = access.New(store, security.Crypto{}, time.Now, 1000, "reopened", store)
	v, c := j.svc.Handle(ctx, 1000, access.Command{Op: "access.login", Owner: "owner", Name: "technical", Password: password})
	if c != access.OK {
		t.Fatal(c)
	}
	j.q = access.Command{Owner: "owner", Session: v.Session, Profile: v.Profile, Activation: v.Activation}
	for _, tc := range []struct {
		company string
		p       inventory.TablePreference
	}{{companyX, x}, {companyY, y}} {
		j.selectCompany(tc.company, access.OK)
		got, c := j.st.GetTablePreference(ctx, tc.company, "equipment")
		if c != inventory.OK || !reflect.DeepEqual(got, &tc.p) {
			t.Fatal("arrangement lost on restart", tc.company, got, c)
		}
	}
}

func TestTableArrangementRejectsOverflowWithoutChangingStoredValue(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	ctx := context.Background()
	p := inventory.TablePreference{Search: "keep", PageSize: 50}
	if c := j.st.SetTablePreference(ctx, companyX, "equipment", p); c != inventory.OK {
		t.Fatal(c)
	}
	bad := p
	bad.PageSize = 30
	huge := p
	huge.Manual = make([]string, 500)
	for i := range huge.Manual {
		huge.Manual[i] = fmt.Sprintf("%03d%s", i, strings.Repeat("x", 190))
	}
	for _, invalid := range []inventory.TablePreference{bad, huge} {
		if c := j.st.SetTablePreference(ctx, companyX, "equipment", invalid); c != inventory.Invalid {
			t.Fatal("invalid preference accepted", c)
		}
	}
	for i := 0; i < 63; i++ {
		if c := j.st.SetTablePreference(ctx, companyX, fmt.Sprintf("table-%d", i), p); c != inventory.OK {
			t.Fatal(i, c)
		}
	}
	if c := j.st.SetTablePreference(ctx, companyX, "overflow", p); c != inventory.Limit {
		t.Fatal("65th table", c)
	}
	got, c := j.st.GetTablePreference(ctx, companyX, "equipment")
	if c != inventory.OK || !reflect.DeepEqual(got, &p) {
		t.Fatal("failed save changed previous arrangement", got, c)
	}
	p.Search = "update at capacity"
	if c := j.st.SetTablePreference(ctx, companyX, "equipment", p); c != inventory.OK {
		t.Fatal("update at capacity", c)
	}
}

func TestEquipmentArrangementSeedsLegacyColumnsOncePerCompany(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.create(companyY, "Y", access.OK)
	j.selectCompany(companyX, access.OK)
	old := []string{"equipment", "block", "cpu"}
	q := j.q
	q.Op = "preferences.columns"
	q.Columns = &old
	j.call(q, access.OK)
	first, c := j.st.GetTablePreference(context.Background(), companyX, "equipment")
	if c != inventory.OK || first == nil || !reflect.DeepEqual(first.Columns, []string{"equipment", "group", "cpu"}) {
		t.Fatal(first, c)
	}
	next := []string{"equipment", "address"}
	q.Columns = &next
	j.call(q, access.OK)
	again, c := j.st.GetTablePreference(context.Background(), companyX, "equipment")
	if c != inventory.OK || !reflect.DeepEqual(first, again) {
		t.Fatal("seed followed later account change", again, c)
	}
	j.selectCompany(companyY, access.OK)
	other, c := j.st.GetTablePreference(context.Background(), companyY, "equipment")
	if c != inventory.OK || other == nil || !reflect.DeepEqual(other.Columns, next) {
		t.Fatal("Y failed to seed its own initial choice", other, c)
	}
}
