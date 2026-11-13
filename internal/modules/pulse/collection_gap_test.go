package pulse_test

import (
	"infradesktop/internal/modules/pulse"
	"testing"
	"time"
)

func TestCollectionGapNeedsMoreThanTwoMinutesWithoutList(t *testing.T) {
	last := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	if gap := pulse.CollectionGap(last, last, last.Add(2*time.Minute)); !gap.IsZero() {
		t.Fatal("one lost 60 s cycle must not become a gap", gap)
	}
	if gap := pulse.CollectionGap(last, last, last.Add(10*time.Minute)); !gap.Equal(last) {
		t.Fatal("ten minutes without list must open a gap at the last success", gap)
	}
	if gap := pulse.CollectionGap(last, time.Time{}, last.Add(time.Hour)); !gap.Equal(last) {
		t.Fatal("service stopped for an hour must open a gap", gap)
	}
	if gap := pulse.CollectionGap(time.Time{}, last, last.Add(time.Hour)); !gap.IsZero() {
		t.Fatal("no previous list means nothing was blind", gap)
	}
}
