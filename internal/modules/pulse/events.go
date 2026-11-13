package pulse

import "time"

type Identity struct{ Firmware, Name, Model string }
type IdentityChange struct{ Kind, Before, After string }

func IdentityChanges(before, after Identity) []IdentityChange {
	values := []IdentityChange{{"firmware",
		before.Firmware,
		after.Firmware},
		{"identity",
			before.Name,
			after.Name},
		{"model",
			before.Model,
			after.Model}}
	out := []IdentityChange{}
	for _, v := range values {
		if v.Before != "" && v.After != "" && v.Before != v.After {
			out = append(out, v)
		}
	}
	return out
}

func PresenceChanged(previous, current DeviceStatus) bool {
	return current.State != Unknown && (previous.Address != current.Address || previous.State != current.State)
}

func PresenceDue(previous, current DeviceStatus, saved, now time.Time) bool {
	return current.State != Unknown && (PresenceChanged(previous, current) || now.Sub(saved) >= time.Minute)
}

// CollectionGap opens a gap only after more than 2 min without a session list:
// one lost 60 s cycle is not a blind interval.
func CollectionGap(last, failed, now time.Time) time.Time {
	if last.IsZero() || now.Sub(last) <= 2*time.Minute {
		return time.Time{}
	}
	if !failed.IsZero() {
		return failed
	}
	return last
}
