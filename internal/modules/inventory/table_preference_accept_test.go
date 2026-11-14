package inventory_test

// Aceite L-TABELA (decisões B1, B2, B3, B5 do Grill do QA 2): a arrumação de cada
// tabela é guardada por empresa e validada antes de gravar. Escrito por Claude; não editar.

import (
	"fmt"
	"infradesktop/internal/modules/inventory"
	"strings"
	"testing"
)

func valida() inventory.TablePreference {
	return inventory.TablePreference{
		Columns:  []string{"state", "role", "block"},
		Filters:  map[string]string{"state": "OFFLINE", "block": "Torre Norte"},
		Search:   "rb",
		Sort:     &inventory.TableSort{Column: "cpu", Desc: true},
		Manual:   []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"},
		PageSize: 50,
	}
}

func TestTablePreferenceAcceptsWhatTheScreensSave(t *testing.T) {
	for _, table := range []string{"equipment", "sessions", "credentials", "ap-clients", "radius-servers"} {
		if !inventory.ValidTablePreference(table, valida()) {
			t.Fatalf("preferência válida recusada para %q", table)
		}
	}
	for _, size := range []int{25, 50, 100, 200} {
		p := valida()
		p.PageSize = size
		if !inventory.ValidTablePreference("equipment", p) {
			t.Fatalf("tamanho de página %d recusado", size)
		}
	}
	vazia := inventory.TablePreference{PageSize: 50}
	if !inventory.ValidTablePreference("equipment", vazia) {
		t.Fatal("preferência sem colunas, filtros, ordem e ordem manual recusada")
	}
}

func TestTablePreferenceRejectsWhatCouldBreakOrBloatTheCompanyDatabase(t *testing.T) {
	manual := make([]string, inventory.MaxDevices+1)
	for i := range manual {
		manual[i] = fmt.Sprintf("%08d-0000-4000-8000-000000000000", i)
	}
	ruins := map[string]func(*inventory.TablePreference){
		"tamanho de página fora da lista":  func(p *inventory.TablePreference) { p.PageSize = 30 },
		"tamanho de página zero":           func(p *inventory.TablePreference) { p.PageSize = 0 },
		"ordem manual acima do inventário": func(p *inventory.TablePreference) { p.Manual = manual },
		"ordem manual com id repetido":     func(p *inventory.TablePreference) { p.Manual = []string{"a", "a"} },
		"busca acima de 160 caracteres":    func(p *inventory.TablePreference) { p.Search = strings.Repeat("x", 161) },
		"coluna repetida":                  func(p *inventory.TablePreference) { p.Columns = []string{"state", "state"} },
		"mais de 32 colunas":               func(p *inventory.TablePreference) { p.Columns = make([]string, 33) },
		"filtro com valor acima de 200": func(p *inventory.TablePreference) {
			p.Filters = map[string]string{"block": strings.Repeat("x", 201)}
		},
		"mais de 16 filtros": func(p *inventory.TablePreference) {
			p.Filters = map[string]string{}
			for i := 0; i < 17; i++ {
				p.Filters[fmt.Sprint("f", i)] = "x"
			}
		},
		"ordenação sem coluna": func(p *inventory.TablePreference) { p.Sort = &inventory.TableSort{} },
		"coluna vazia":         func(p *inventory.TablePreference) { p.Columns = []string{""} },
	}
	for nome, estraga := range ruins {
		p := valida()
		estraga(&p)
		if inventory.ValidTablePreference("equipment", p) {
			t.Fatalf("aceitou: %s", nome)
		}
	}
	for _, table := range []string{"", "Equipment", "equip ment", "../x", strings.Repeat("a", 49)} {
		if inventory.ValidTablePreference(table, valida()) {
			t.Fatalf("aceitou nome de tabela inválido %q", table)
		}
	}
}
