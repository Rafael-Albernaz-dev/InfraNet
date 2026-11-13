package pppoe

import (
	"infradesktop/internal/modules/equipment"
	"testing"
)

func TestL2TPSecretAppearsOnlyInItsOwnTable(t *testing.T) {
	catalog := []equipment.PPPEntry{
		{ID: "1", Kind: "secret", Name: "vpn", Service: "l2tp"},
		{ID: "2", Kind: "secret", Name: "ppp", Service: "pppoe"},
	}
	EnrichPPPEntries(catalog, nil)
	regular := filterPPP(catalog, &equipment.PPPQuery{Screen: "secrets"})
	vpn := filterPPP(catalog, &equipment.PPPQuery{Screen: "l2tp"})
	if len(regular) != 1 || regular[0].Name != "ppp" || len(vpn) != 1 || vpn[0].Name != "vpn" {
		t.Fatal(regular, vpn)
	}
}
