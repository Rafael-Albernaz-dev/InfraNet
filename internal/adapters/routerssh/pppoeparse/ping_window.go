package pppoeparse

import (
	"infradesktop/internal/modules/equipment"
	"strconv"
	"strings"
)

// ParsePingWindowLine accepts only RouterOS ICMP rows for the validated target.
// Summary lines are generated from these rows, not trusted as arbitrary text.
func ParsePingWindowLine(line, address string) (equipment.PDPingProbe, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[1] != address {
		return equipment.PDPingProbe{}, false
	}
	st := &pdPingParseState{count: 1000000, address: address,
		result: &equipment.PDPingResult{}, sequences: map[int]bool{}, header: true}
	if !st.parseProbeLine(line) || len(st.result.Probes) != 1 {
		return equipment.PDPingProbe{}, false
	}
	probe := st.result.Probes[0]
	seq, err := strconv.Atoi(fields[0])
	return probe, err == nil && seq == probe.Sequence
}
