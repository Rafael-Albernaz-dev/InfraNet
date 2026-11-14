package radio

import (
	"net/netip"
	"time"
)

type EnlaceState string

const (
	EnlaceFound  EnlaceState = "found"
	EnlaceTwo    EnlaceState = "two"
	EnlaceNone   EnlaceState = "none"
	EnlaceUnread EnlaceState = "unread"
)

type APClient struct{ MAC, Signal, Interface, BSSID string }
type APSeen struct {
	AP      string
	Failed  bool
	Clients []APClient
}
type EnlaceChoice struct{ AP, Signal, MAC, Interface, BSSID string }
type EnlaceResult struct {
	State   EnlaceState
	AP      string
	Choices []EnlaceChoice
	Failed  []string
}

// EnlaceMatch only considers the linked APs supplied by the caller. IP and names
// are deliberately not evidence of a wireless association.
func EnlaceMatch(caller string, aps []APSeen) EnlaceResult {
	result := EnlaceResult{State: EnlaceNone}
	mac, valid := RadioMAC(caller)
	read := false
	for _, ap := range aps {
		if ap.Failed {
			result.Failed = append(result.Failed, ap.AP)
			continue
		}
		read = true
		if !valid {
			continue
		}
		for _, client := range ap.Clients {
			observed, ok := RadioMAC(client.MAC)
			if ok && mac == observed {
				result.Choices = append(result.Choices, EnlaceChoice{ap.AP, client.Signal, mac, client.Interface, client.BSSID})
				break
			}
		}
	}
	if !valid {
		return result
	}
	switch {
	case !read:
		result.State = EnlaceUnread
	case len(result.Choices) == 1:
		result.State, result.AP = EnlaceFound, result.Choices[0].AP
	case len(result.Choices) > 1:
		result.State = EnlaceTwo
	}
	return result
}

func FollowLogin(current string, row PPRow) (string, bool) {
	ip, err := netip.ParseAddr(row.Address)
	if row.State != "ONLINE" || err != nil || ip.IsUnspecified() || ip.IsMulticast() {
		return current, false
	}
	return row.Address, current != row.Address
}

func RondaDue(id string, last, now time.Time) bool {
	if last.IsZero() {
		return true
	}
	hash := uint32(2166136261)
	for i := 0; i < len(id); i++ {
		hash ^= uint32(id[i])
		hash *= 16777619
	}
	interval := 15*time.Minute + time.Duration(hash%uint32(15*time.Minute/time.Second+1))*time.Second
	return now.Sub(last) >= interval
}

func enlaceInterface(row PPRow) string {
	if row.Context == nil || row.Context.Association != "associated" {
		return ""
	}
	return row.Context.Interface
}
