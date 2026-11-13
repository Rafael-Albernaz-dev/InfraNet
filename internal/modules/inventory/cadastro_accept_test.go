package inventory_test

// Aceite L-CADASTRO (decisões C1, C4 e C6 do Grill do QA 2; TELAS D18 e D27):
// chave guardada sozinha só no primeiro contato, nome = identidade lida e grupos com subgrupos.
// Escrito por Claude; não editar.

import (
	"fmt"
	"infradesktop/internal/modules/inventory"
	"strings"
	"testing"
)

const (
	regiao = "11111111-1111-4111-8111-111111111111"
	torre  = "22222222-2222-4222-8222-222222222222"
	painel = "33333333-3333-4333-8333-333333333333"
	outra  = "44444444-4444-4444-8444-444444444444"
	quinto = "55555555-5555-4555-8555-555555555555"
	sexto  = "77777777-7777-4777-8777-777777777777"
)

func arvore() []inventory.Group {
	return []inventory.Group{
		{ID: regiao, Name: "Região Norte"},
		{ID: torre, Parent: regiao, Name: "Torre Ribeirão"},
		{ID: painel, Parent: torre, Name: "Painel 04"},
		{ID: outra, Name: "Região Sul"},
	}
}

func TestFirstContactStoresKeyOnlyWhenNoneIsSaved(t *testing.T) {
	const salva, nova = "SHA256:salva", "SHA256:nova"
	if got, code := inventory.FirstContactKey("", nova); got != nova || code != inventory.OK {
		t.Fatalf("primeiro contato: quer %q OK, veio %q %v", nova, got, code)
	}
	if got, code := inventory.FirstContactKey(salva, salva); got != salva || code != inventory.OK {
		t.Fatalf("mesma chave: quer %q OK, veio %q %v", salva, got, code)
	}
	if got, code := inventory.FirstContactKey(salva, nova); got != salva || code != inventory.TrustChanged {
		t.Fatalf("chave alterada nunca substitui a guardada: quer %q TrustChanged, veio %q %v", salva, got, code)
	}
	if _, code := inventory.FirstContactKey("", ""); code == inventory.OK {
		t.Fatal("contato sem chave apresentada não pode ser aceito")
	}
}

func TestDisplayNameFollowsIdentityOrShowsAddress(t *testing.T) {
	devices := []inventory.Device{
		{ID: regiao, Address: "10.0.0.1", Identity: "CCR-CENTRO"},
		{ID: torre, Address: "10.0.0.2", Identity: "MikroTik"},
		{ID: painel, Address: "10.0.0.3", Identity: "MikroTik"},
		{ID: outra, Address: "10.0.0.4", Name: "Nome digitado antes", Identity: ""},
	}
	got := inventory.DisplayNames(devices)
	want := map[string]string{
		regiao: "CCR-CENTRO",
		torre:  "MikroTik (10.0.0.2)",
		painel: "MikroTik (10.0.0.3)",
		outra:  "10.0.0.4",
	}
	for id, name := range want {
		if got[id] != name {
			t.Fatalf("nome de %s: quer %q, veio %q", id, name, got[id])
		}
	}
}

func TestGroupsFormATreeAndFilterIncludesSubgroups(t *testing.T) {
	g := arvore()
	if !inventory.ValidGroups(g) {
		t.Fatal("árvore válida recusada")
	}
	if p := inventory.GroupPath(g, painel); p != "Região Norte › Torre Ribeirão › Painel 04" {
		t.Fatalf("caminho do painel: %q", p)
	}
	for _, id := range []string{regiao, torre, painel} {
		if !inventory.GroupIncludes(g, regiao, id) {
			t.Fatalf("filtrar Região Norte deve incluir %s", id)
		}
	}
	if inventory.GroupIncludes(g, regiao, outra) || inventory.GroupIncludes(g, painel, torre) {
		t.Fatal("filtro de grupo não pode incluir irmão nem ancestral")
	}
	mesmoNome := append(arvore(), inventory.Group{ID: quinto, Parent: outra, Name: "Torre Ribeirão"})
	if !inventory.ValidGroups(mesmoNome) {
		t.Fatal("mesmo nome sob pais diferentes deve valer")
	}
}

func TestGroupsRejectWhatBreaksTheTree(t *testing.T) {
	extra := func(parent, name string) []inventory.Group {
		return append(arvore(), inventory.Group{ID: quinto, Parent: parent, Name: name})
	}
	ruins := map[string][]inventory.Group{
		"irmão com o mesmo nome, sem diferenciar maiúsculas": extra(regiao, "torre ribeirão"),
		"pai inexistente": extra("66666666-6666-4666-8666-666666666666", "Solto"),
		"quinto nível": append(extra(painel, "Nível 4"),
			inventory.Group{ID: sexto, Parent: quinto, Name: "Nível 5"}),
		"nome vazio":                 extra(regiao, ""),
		"nome com espaço nas pontas": extra(regiao, " Torre "),
		"nome acima de 80":           extra(regiao, strings.Repeat("x", 81)),
		"nome com controle":          extra(regiao, "Torre\tSul"),
		"nome com separador ›":       extra(regiao, "A › B"),
		"nome com separador >":       extra(regiao, "A > B"),
		"ciclo": {
			{ID: regiao, Parent: torre, Name: "A"},
			{ID: torre, Parent: regiao, Name: "B"},
		},
		"id repetido": {
			{ID: regiao, Name: "A"},
			{ID: regiao, Name: "B"},
		},
	}
	for nome, g := range ruins {
		if inventory.ValidGroups(g) {
			t.Fatalf("aceitou árvore inválida: %s", nome)
		}
	}
	muitos := make([]inventory.Group, 501)
	for i := range muitos {
		muitos[i] = inventory.Group{ID: fmt.Sprintf("%08d-0000-4000-8000-000000000000", i), Name: fmt.Sprintf("G%d", i)}
	}
	if inventory.ValidGroups(muitos) {
		t.Fatal("aceitou mais de 500 grupos")
	}
}

func TestSpreadsheetGroupPath(t *testing.T) {
	boas := map[string][]string{
		"Região Norte › Torre Ribeirão": {"Região Norte", "Torre Ribeirão"},
		"Região Norte > Torre Ribeirão": {"Região Norte", "Torre Ribeirão"},
		"  Região Norte  ":              {"Região Norte"},
		"":                              nil,
	}
	for in, want := range boas {
		got, ok := inventory.SplitGroupPath(in)
		if !ok || strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("SplitGroupPath(%q) = %q %v, quer %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"A ›  › B", "A › B › C › D › E", "A >", strings.Repeat("x", 81)} {
		if _, ok := inventory.SplitGroupPath(in); ok {
			t.Fatalf("SplitGroupPath(%q) aceitou caminho inválido", in)
		}
	}
}
