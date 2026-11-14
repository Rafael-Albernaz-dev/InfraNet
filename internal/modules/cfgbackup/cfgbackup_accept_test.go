package cfgbackup_test

// Aceite L-CFG (Grill do QA 2, bloco G; TELAS §4.5): a cópia só conta como mudança quando
// a configuração muda de fato, a comparação mostra o que saiu e o que entrou, e linha com
// senha aparece mascarada na comparação. Escrito por Claude; não editar.

import (
	"infradesktop/internal/modules/cfgbackup"
	"strings"
	"testing"
)

const cfgExport6 = "# nov/14/2026 03:10:22 by RouterOS 6.49.10\n# software id = ABCD-1234\n" +
	"/ip address\nadd address=192.0.2.1/24 interface=ether1\n"
const cfgExport6Later = "# nov/15/2026 03:10:09 by RouterOS 6.49.10\n# software id = ABCD-1234\n" +
	"/ip address\nadd address=192.0.2.1/24 interface=ether1\n"
const cfgExport7 = "# 2026-11-14 03:10:22 by RouterOS 7.16.1\n# software id = ABCD-1234\n/system identity\nset name=ccr\n"

func TestNormalizeIgnoresOnlyTheExportTimestamp(t *testing.T) {
	if cfgbackup.Normalize(cfgExport6) != cfgbackup.Normalize(cfgExport6Later) {
		t.Fatal("export igual em outro dia não pode virar versão nova")
	}
	if strings.Contains(cfgbackup.Normalize(cfgExport7), "03:10:22") {
		t.Fatal("linha de data do RouterOS 7 deveria sair da comparação")
	}
	if !strings.Contains(cfgbackup.Normalize(cfgExport6), "# software id = ABCD-1234") {
		t.Fatal("só a linha de data sai; o resto do cabeçalho fica")
	}
}

func TestDiffCountsAddedAndRemovedLines(t *testing.T) {
	change := cfgbackup.Diff("a\nb\nc\n", "a\nc\nd\n")
	if change.Added != 1 || change.Removed != 1 {
		t.Fatalf("quer +1 −1, veio +%d −%d", change.Added, change.Removed)
	}
	kinds := []string{}
	for _, l := range change.Lines {
		kinds = append(kinds, l.Kind+":"+l.Text)
	}
	if strings.Join(kinds, ",") != "same:a,remove:b,same:c,add:d" {
		t.Fatalf("diferença fora da ordem: %v", kinds)
	}
}

func TestMaskHidesSecretValuesOnly(t *testing.T) {
	cases := map[string]string{
		`add name=ana password=s3nh4 profile=p100`:      `add name=ana password=******** profile=p100`,
		`set wpa2-pre-shared-key="minha chave" mode=ap`: `set wpa2-pre-shared-key=******** mode=ap`,
		`users.1.password=$1$abc$def`:                   `users.1.password=********`,
		`wpa.1.psk=segredo`:                             `wpa.1.psk=********`,
		`add address=192.0.2.1/24 interface=ether1`:     `add address=192.0.2.1/24 interface=ether1`,
	}
	for in, want := range cases {
		if got := cfgbackup.Mask(in); got != want {
			t.Fatalf("Mask(%q) = %q, quer %q", in, got, want)
		}
	}
}

func TestDiffMasksPasswordChanges(t *testing.T) {
	change := cfgbackup.Diff("add name=ana password=velha\n", "add name=ana password=nova\n")
	if change.Added != 1 || change.Removed != 1 {
		t.Fatalf("troca de senha é mudança: +%d −%d", change.Added, change.Removed)
	}
	for _, l := range change.Lines {
		if strings.Contains(l.Text, "velha") || strings.Contains(l.Text, "nova") {
			t.Fatalf("senha apareceu na comparação: %q", l.Text)
		}
	}
}
