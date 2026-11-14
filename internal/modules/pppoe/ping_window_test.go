package pppoe

import (
	"infradesktop/internal/modules/equipment"
	"testing"
	"time"
)

func TestPingWindowStopsAtFifteenMinutesByDomainClock(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	window := PingWindow{Started: now}
	if window.Expired(now.Add(15*time.Minute-time.Nanosecond)) || !window.Expired(now.Add(15*time.Minute)) {
		t.Fatal("15 minute boundary not enforced")
	}
	if PPReadBudget(equipment.PingWindowScope) != 15*time.Minute {
		t.Fatal("worker deadline differs from terminal budget")
	}
}
