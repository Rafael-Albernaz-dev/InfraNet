package pppoe

import (
	"testing"
	"time"
)

func TestNewSessionsRequestOneContextReadingPerFiveMinutes(t *testing.T) {
	rows := make([]PPRow, 30)
	for i := range rows {
		rows[i].Login = "new"
	}
	now := time.Unix(1000, 0)
	if !NeedsSessionContext(rows, time.Time{}, now) {
		t.Fatal("new sessions did not request context")
	}
	for _, age := range []time.Duration{time.Minute, 4 * time.Minute, 5*time.Minute - time.Nanosecond} {
		if NeedsSessionContext(rows, now, now.Add(age)) {
			t.Fatal("repeated full context before five minutes", age)
		}
	}
	if !NeedsSessionContext(rows, now, now.Add(5*time.Minute)) {
		t.Fatal("context retry never became due")
	}
	for i := range rows {
		rows[i].Context = &PPContext{SecretState: "none"}
	}
	if NeedsSessionContext(rows, now, now.Add(10*time.Minute)) {
		t.Fatal("complete context re-read without missing session")
	}
}
