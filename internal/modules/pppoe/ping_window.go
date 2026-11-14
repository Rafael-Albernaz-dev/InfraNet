package pppoe

import (
	"fmt"
	"infradesktop/internal/modules/equipment"
	"time"
)

// PingWindow accumulates the terminal transcript and summary in bounded memory.
// The daemon deadline uses the same domain budget, independent of UI lifetime.
type PingWindow struct {
	Started                 time.Time
	Rows                    []PPReadRow
	sent, received          int
	total, minimum, maximum int64
}

func (p *PingWindow) Expired(now time.Time) bool {
	return now.Sub(p.Started) >= equipment.PingWindowBudget
}
func (p *PingWindow) Add(line string, probe PDPingProbe) Code {
	if p.sent >= 1000 || probe.Sequence != p.sent {
		return Partial
	}
	p.sent++
	if probe.RTT != nil && (probe.Status == "" || probe.Status == "echo reply") {
		p.received++
		p.total += *probe.RTT
		if p.received == 1 || *probe.RTT < p.minimum {
			p.minimum = *probe.RTT
		}
		p.maximum = max(p.maximum, *probe.RTT)
	}
	p.Rows = append(p.Rows, PPReadRow{Kind: equipment.PingWindowScope, Value: line})
	return OK
}
func (p *PingWindow) Result(now time.Time) *PPReadResult {
	loss := 0.0
	if p.sent > 0 {
		loss = float64(p.sent-p.received) * 100 / float64(p.sent)
	}
	summary := fmt.Sprintf("sent=%d received=%d packet-loss=%.1f%%", p.sent, p.received, loss)
	if p.received > 0 {
		summary += fmt.Sprintf(" min-rtt=%s avg-rtt=%s max-rtt=%s",
			time.Duration(p.minimum), time.Duration(p.total/int64(p.received)), time.Duration(p.maximum))
	}
	return &PPReadResult{Scope: equipment.PingWindowScope, Started: PPTime(p.Started), Ended: PPTime(now),
		Duration: now.Sub(p.Started).Nanoseconds(), Rows: append([]PPReadRow(nil), p.Rows...), PingSummary: summary}
}
