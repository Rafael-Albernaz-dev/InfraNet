package inventory_test

import (
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"testing"
	"time"
)

func TestFirstContactEligibilityAndTenMinuteBudget(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	base := equipment.Device{Active: true, CredentialID: "credential", Revision: 7}
	type scenario struct {
		name     string
		device   equipment.Device
		revision uint64
		due      bool
	}
	cases := []scenario{
		{"eligible", base, 3, true},
		{"inactive", equipment.Device{CredentialID: "credential"}, 3, false},
		{"without credential", equipment.Device{Active: true}, 3, false},
		{"already pinned", equipment.Device{Active: true, CredentialID: "credential", Fingerprint: "saved"}, 3, false},
	}
	for _, age := range []time.Duration{0, 9*time.Minute + 59*time.Second, 10 * time.Minute} {
		d := base
		d.LastAttempt = &equipment.Attempt{Code: equipment.Unavailable, At: equipment.PPTime(now.Add(-age)), DeviceRevision: 7,
			CredentialRevision: 3}
		cases = append(cases, scenario{age.String(), d, 3, age >= 10*time.Minute})
	}
	refused := base
	refused.LastAttempt = &equipment.Attempt{Code: equipment.RadioCodeRemoteAuthFailed,
		At: equipment.PPTime(now.Add(-time.Hour)), DeviceRevision: 7, CredentialRevision: 3}
	cases = append(cases, scenario{"refused", refused, 3, false})
	revised := refused
	revised.Revision++
	cases = append(cases, scenario{"device changed", revised, 3, true})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if inventory.ContactDue(tc.device, tc.revision, now) != tc.due {
				t.Fatal("eligibility/budget mismatch", tc.name)
			}
		})
	}
}
