package routerssh_test

import (
	"context"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"testing"
)

func TestFirstContactReadsRouterOSAndRejectsChangedKeyBeforePassword(t *testing.T) {
	peer, d := livePeer(t, sshfixture.Config{Family: "v7", ServerVersion: "SSH-2.0-ROSSSH"})
	d.Fingerprint = ""
	result, code := (routerssh.Adapter{}).FirstContact(context.Background(), d, "reader", sshfixture.Password,
		func() equipment.Code { return equipment.OK })
	if code != equipment.OK || result == nil || result.Observation == nil || result.Observation.Identity == "" {
		t.Fatal(code, result)
	}
	if result.Fingerprint != peer.Fingerprint() || peer.Auth.Load() != 1 {
		t.Fatal("missing pin or authentication", peer.Auth.Load())
	}
	d.Fingerprint = "SHA256:changed"
	result, code = (routerssh.Adapter{}).FirstContact(context.Background(), d, "reader", sshfixture.Password,
		func() equipment.Code { return equipment.OK })
	if code != equipment.TrustChanged || result != nil || peer.Auth.Load() != 1 {
		t.Fatal("changed key received password", code)
	}
}
