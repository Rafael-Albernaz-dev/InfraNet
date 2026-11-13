package pulse

import (
	"context"
	"time"
)

const HistoryRetention = 90 * 24 * time.Hour

func BootTime(at time.Time, uptime time.Duration) time.Time { return at.Add(-uptime) }
func Rebooted(previous, current time.Time) bool {
	return !previous.IsZero() && current.Sub(previous) > time.Minute
}

func RebootCounts(boots []time.Time, now time.Time) (last time.Time, seven, thirty int) {
	for _, boot := range boots {
		if boot.After(now) {
			continue
		}
		if boot.After(last) {
			last = boot
		}
		if !boot.Before(now.Add(-7 * 24 * time.Hour)) {
			seven++
		}
		if !boot.Before(now.Add(-30 * 24 * time.Hour)) {
			thirty++
		}
	}
	return
}

type PresenceStore interface {
	LoadPresence(context.Context, string) (map[string]DeviceStatus, error)
	SavePresence(context.Context, string, map[string]DeviceStatus, time.Time) error
}

// Restore retains the last confirmed reply, never a stale ONLINE claim after restart.
func (t *Tracker) Restore(values map[string]DeviceStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, value := range values {
		d := t.devices[id]
		if d != nil && d.target.Address == value.Address && d.status.LastReply.IsZero() {
			d.status.LastReply = value.LastReply
		}
	}
}
