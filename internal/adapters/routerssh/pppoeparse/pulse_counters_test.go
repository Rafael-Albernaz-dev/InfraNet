package pppoeparse

import (
	"fmt"
	"infradesktop/internal/modules/equipment"
	"os"
	"strings"
	"testing"
)

func TestPageCountersAreBoundedAndGoldenPreservesExactBytes(t *testing.T) {
	body, err := os.ReadFile("testdata/pulse_counters_v7.txt")
	if err != nil {
		t.Fatal(err)
	}
	counters, code := ParsePulseCounters(string(body))
	if code != equipment.OK ||
		len(counters) != 2 ||
		counters[0].Name != "<pppoe-fixture-1>" ||
		*counters[0].TX != 87654321 ||
		*counters[1].RX != 9007199254740993 {
		t.Fatal(code, counters)
	}
	rows := []equipment.PPRow{}
	for i := 0; i < 200; i++ {
		rows = append(rows, equipment.PPRow{Login: fmt.Sprintf("fixture-%d", i)})
	}
	command, ok := BuildPPBandCommand(rows)
	if !ok || !strings.Contains(command, "user=\"fixture-199\"") {
		t.Fatal("visible page rejected")
	}
	// ignored: Only the status or membership is needed in this check.
	if _, ok = BuildPPBandCommand(append(rows, equipment.PPRow{Login: "overflow"})); ok {
		t.Fatal("page limit bypassed")
	}
	if strings.Count(PulseCountersCommand,
		"/interface find") != 1 ||
		strings.Contains(PulseCountersCommand,
			"monitor") ||
		strings.Contains(PulseCountersCommand,
			"find where name") {
		t.Fatal("quadratic or per-row monitor in background")
	}
}
