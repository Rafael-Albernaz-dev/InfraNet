package storage

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pppoe"
	"testing"
	"time"
)

func TestCatalogClientIsLocalSearchableAndLatestReadingReplacesPrevious(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	o := ppSynthetic(t, j, target, 1, "ana")
	result := &equipment.PPReadResult{Scope: equipment.PPPCatalogScope, Ended: equipment.PPTime(time.Now()), Rows: []equipment.PPReadRow{
		{Kind: "secret", Entry: &equipment.PPPEntry{ID: "*1", Kind: "secret", Name: "ana", Client: pppoe.ClientName("SGP: Ana Maria")}},
	}}
	if code := j.st.SavePPPCatalog(ctx, companyX, target.Device, target.Credential.Metadata.Revision, result); code != equipment.OK {
		t.Fatal(code)
	}
	page, code := j.st.PPPage(ctx, companyX, equipment.PPInput{
		DeviceID: target.Device.ID, ObservationID: o.ID, Query: &equipment.PPQuery{Search: "Ana Maria"},
	})
	if code != equipment.OK || page.Total != 1 || len(page.Rows) != 1 || page.Rows[0].Client != "Ana Maria" {
		t.Fatal(code, page)
	}
	detail, code := j.st.PPSession(ctx, companyX, equipment.PPInput{DeviceID: target.Device.ID, ObservationID: o.ID, RowID: page.Rows[0].ID})
	if code != equipment.OK || detail.Rows[0].Client != "Ana Maria" {
		t.Fatal("detail lost local client", code, detail)
	}
	result.Rows[0].Entry.Client = "Bia"
	if code = j.st.SavePPPCatalog(ctx, companyX, target.Device, target.Credential.Metadata.Revision, result); code != equipment.OK {
		t.Fatal(code)
	}
	entries, _, _, code := j.st.ReadPPPCatalog(ctx, companyX, target.Device.ID)
	if code != equipment.OK || len(entries) != 1 || entries[0].Client != "Bia" {
		t.Fatal("catalog not replaced", code, entries)
	}
}

func TestTodayDisconnectCountExcludesCollectionGapsAndYesterday(t *testing.T) {
	j, target := ppStoreSetup(t)
	ctx := context.Background()
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for _, row := range []struct {
		kind string
		at   time.Time
	}{{"session", now}, {"gap", now}, {"session", start.Add(-time.Second)}} {
		// ignored: Only the SQL error is needed for this synthetic history fixture.
		_, err := j.st.unit.Exec(`INSERT INTO session_history(device_id,login,session_key,kind,connected_at,disconnected_at,reason,row_json)
   VALUES(?,'ana','key',?,'',?,'','{}')`, target.Device.ID, row.kind, equipment.PPTime(row.at))
		if err != nil {
			t.Fatal(err)
		}
	}
	n, code := j.st.PPPDropsToday(ctx, companyX, target.Device.ID, equipment.PPTime(start))
	if code != equipment.OK || n != 1 {
		t.Fatal("gap/yesterday counted as today's disconnect", code, n)
	}
}
