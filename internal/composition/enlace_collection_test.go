package composition

import (
	"context"
	"fmt"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/radio"
	"testing"
	"time"
)

func TestTwoThousandEnlaceCPEsStayOutOfPresenceAndMinuteReaders(t *testing.T) {
	devices := make([]equipment.Device, 0, 2001)
	for i := 0; i < 2000; i++ {
		devices = append(devices, equipment.Device{ID: fmt.Sprint(i), Role: equipment.RoleCPE, Active: true, Address: "192.0.2.2",
			CredentialID: "c", Transport: equipment.TransportSSH, Fingerprint: "saved", Revision: 1,
			CPELogin: &equipment.CPELogin{Login: fmt.Sprint(i), Origin: "enlace"}})
	}
	devices = append(devices, equipment.Device{ID: "concentrator", Role: equipment.RoleConcentrator, Active: true,
		Address: "192.0.2.1", CredentialID: "c", Transport: equipment.TransportSSH, Fingerprint: "saved", Revision: 1})
	inv, store := liveTestInventory(devices...)
	_, reads, ok := inv.ReadTargets(context.Background())
	if !ok || len(reads) != 1 || reads[0].ID != "concentrator" {
		t.Fatal("minute reader admitted CPEs", len(reads))
	}
	presence := pulseInventory{access: inv.access, store: store}
	_, targets, ok := presence.Targets(context.Background())
	if !ok || len(targets) != 1 || targets[0].ID != "concentrator" {
		t.Fatal("presence admitted CPEs", len(targets))
	}
	now := time.Now()
	early := 0
	for _, d := range devices[:2000] {
		if radio.RondaDue(d.ID, now.Add(-14*time.Minute), now) || !radio.RondaDue(d.ID, now.Add(-30*time.Minute), now) {
			t.Fatal("CPE outside 15-30 minute ronda", d.ID)
		}
		if radio.RondaDue(d.ID, now.Add(-20*time.Minute), now) {
			early++
		}
	}
	if early == 0 || early == 2000 {
		t.Fatal("cohort not spread", early)
	}
}
