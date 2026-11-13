package pppoeparse

import (
	"fmt"
	"infradesktop/internal/modules/equipment"
	"os"
	"strings"
	"testing"
)

func TestCatalogReadsThreeThousandSecretsWithoutPasswordsAndCapsAtFourThousand(t *testing.T) {
	if strings.Contains(PPPCatalogCommand, "password") || strings.Contains(PPPCatalogCommand, " print") {
		t.Fatal("catalog requests a secret map or password")
	}
	if !strings.Contains(PPPCatalogCommand, "$count<4096") || !strings.Contains(PPPCatalogCommand, "on-error={}") {
		t.Fatal("catalog is not bounded or optional properties can abort it")
	}
	for _, n := range []int{3000, 4096} {
		var text strings.Builder
		text.WriteString("P\t1\n")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&text, "secret\t*%d\t=user-%d\t=pppoe\t=100M\t=true\t=SGP: Ana\t=\t=\t?\t=\t=\t=\n", i, i)
		}
		fmt.Fprintf(&text, "Z\t%d\t%t\n", n, n == 4096)
		rows, limited, code := ParsePPPCatalog(text.String())
		if code != equipment.OK || len(rows) != n || limited != (n == 4096) || rows[0].Entry.Client != "Ana" {
			t.Fatal(code, len(rows), limited)
		}
	}
}

func TestCatalogAbsentLastLogoutRemainsUnknown(t *testing.T) {
	body, err := os.ReadFile("testdata/ppp_catalog_v7.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, limited, code := ParsePPPCatalog(string(body))
	if code != equipment.OK || limited || len(rows) != 4 {
		t.Fatal(code, limited, len(rows))
	}
	if rows[0].Entry.LastOut != "" || rows[1].Entry.LastOut != "nov/13/2026 09:00:00" ||
		rows[3].Entry.Invalid == nil || !*rows[3].Entry.Invalid {
		t.Fatal("optional logout or invalid server state was invented/lost", rows)
	}
}
