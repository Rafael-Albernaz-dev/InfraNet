package pppoe

import (
	"testing"
	"time"
)

func TestCounterMeanCoversAllSessionsAndKeepsNewOrResetUnknown(t *testing.T) {
	old := []PPTunnel{}
	now := []PPTunnel{}
	rows := []PPRow{}
	for _, name := range []string{"positive", "reset", "new"} {
		r := PPRow{Login: name}
		rows = append(rows, r)
		rx, tx := uint64(100), uint64(200)
		afterRX, afterTX := uint64(160), uint64(320)
		if name == "reset" {
			afterRX, afterTX = 1, 1
		}
		before := PPTunnel{ID: name, Name: "<pppoe-" + name + ">", Login: name, Caller: "MAC", Address: "192.0.2.1", RX: &rx, TX: &tx,
			Sample: PPInterval{Known: true,
				Min: 0,
				Max: 0},
			Uptime: PPInterval{Known: true,
				Min: int64(time.Hour),
				Max: int64(time.Hour)},
			RouterUptime: PPInterval{Known: true,
				Min: int64(2 * time.Hour),
				Max: int64(2 * time.Hour)}}
		after := before
		after.RX, after.TX = &afterRX, &afterTX
		after.Sample = PPInterval{Known: true, Min: int64(time.Minute), Max: int64(time.Minute)}
		after.Uptime.Min += int64(time.Minute)
		after.Uptime.Max = after.Uptime.Min
		after.RouterUptime.Min += int64(time.Minute)
		after.RouterUptime.Max = after.RouterUptime.Min
		if name != "new" {
			old = append(old, before)
		}
		now = append(now, after)
	}
	AverageSessions(rows, old, now)
	if rows[0].Download == nil ||
		rows[0].Download.Min > 16 ||
		rows[0].Download.Max < 16 ||
		rows[0].Upload == nil ||
		rows[0].Upload.Min > 8 ||
		rows[0].Upload.Max < 8 {
		t.Fatal(rows[0])
	}
	if rows[0].Context == nil {
		t.Fatal("known interface was not associated")
	}
	if rows[0].Context.Service != "" {
		t.Fatal("protocol was invented as server service", rows[0].Context)
	}
	for _, r := range rows[1:] {
		if r.Download != nil || r.Upload != nil {
			t.Fatal("unknown became zero", r)
		}
	}
}
